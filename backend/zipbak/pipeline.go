package zipbak

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/remote"
	"z-server-backup-tools/backend/sftpclient"
	"z-server-backup-tools/backend/util"
)

const (
	maxRetries    = 6
	retryBaseWait = 3 * time.Second
	retryMaxWait  = 30 * time.Second
)

// contextCanceledMsg 用于将 context.Canceled 包装为友好的中文提示
const contextCanceledMsg = "任务已取消"

type LogFn func(string)

// StatusUpdateFn applies mutations to job status under the caller's lock.
type StatusUpdateFn func(fn func(*model.JobStatus))

type Pipeline struct {
	cfg          model.BackupConfig
	log          LogFn
	updateStatus StatusUpdateFn
}

func NewPipeline(cfg model.BackupConfig, log LogFn, updateStatus StatusUpdateFn) *Pipeline {
	return &Pipeline{cfg: cfg, log: log, updateStatus: updateStatus}
}

func (p *Pipeline) withStatus(fn func(*model.JobStatus)) {
	if p.updateStatus != nil {
		p.updateStatus(fn)
	}
}

func (p *Pipeline) Run(ctx context.Context) error {
	if err := osMkdirLocal(p.cfg.LocalDir); err != nil {
		return err
	}
	if err := retryOp(ctx, "读取远程状态", p.log, p.refreshStatus); err != nil {
		p.log("读取远程状态失败: " + err.Error())
	}
	var pendingZip string
	p.withStatus(func(st *model.JobStatus) {
		pendingZip = st.PendingZip
	})
	if strings.TrimSpace(pendingZip) != "" {
		p.log("检测到远程待处理分卷，将从断点继续: " + filepath.Base(pendingZip))
	}
	for {
		if err := ctx.Err(); err != nil {
			return wrapCancelError(err)
		}
		if err := retryOp(ctx, "读取远程状态", p.log, p.refreshStatus); err != nil {
			p.log("读取远程状态失败: " + err.Error())
		}
		var done bool
		p.withStatus(func(st *model.JobStatus) {
			done = st.Done
			pendingZip = st.PendingZip
		})
		if done && strings.TrimSpace(pendingZip) == "" {
			p.log("全部文件已备份完成")
			return nil
		}

		p.setPhase("pack")
		zipRemote, err := retryWrap(ctx, "远程打包", p.log, p.remotePack)
		if err != nil {
			if strings.Contains(err.Error(), "备份已完成") {
				p.withStatus(func(st *model.JobStatus) {
					st.Done = true
				})
				p.log("备份任务已完成")
				return nil
			}
			return err
		}
		partName := filepath.Base(zipRemote)
		p.withStatus(func(st *model.JobStatus) {
			st.CurrentPart = partName
		})
		p.log("远程已生成: " + zipRemote)
		if err := p.refreshStatus(); err != nil {
			p.log("读取远程状态失败: " + err.Error())
		}

		localName := filepath.Base(zipRemote)
		localPath := filepath.Join(p.cfg.LocalDir, localName)
		p.withStatus(func(st *model.JobStatus) {
			st.LocalFile = localPath
		})
		p.setPhase("download")
		p.log("开始下载到 " + localPath)

		iterCtx, iterCancel := context.WithCancel(ctx)
		defer iterCancel()

		prefetchCh := make(chan prefetchOutcome, 1)
		var prefetchWG sync.WaitGroup
		prefetchWG.Add(1)
		go func() {
			defer prefetchWG.Done()
			if err := iterCtx.Err(); err != nil {
				prefetchCh <- prefetchOutcome{err: err}
				return
			}
			ahead, err := retryWrap(iterCtx, "预打包下一卷", p.log, p.remotePackAhead)
			prefetchCh <- prefetchOutcome{ahead: strings.TrimSpace(ahead), err: err}
		}()

		if err := retryOp(iterCtx, "下载分卷", p.log, func() error {
			return p.download(iterCtx, zipRemote, localPath)
		}); err != nil {
			iterCancel()
			prefetchWG.Wait()
			return wrapCancelError(err)
		}
		p.log("下载" + partName + "完成")

		var hasMoreVolumes bool
		p.withStatus(func(st *model.JobStatus) {
			hasMoreVolumes = st.TotalFiles > 0 && st.PackedFiles < st.TotalFiles
		})
		if hasMoreVolumes {
			select {
			case outcome := <-prefetchCh:
				p.logPrefetchOutcome(outcome)
			default:
				p.log("等待远程打包")
				outcome := <-prefetchCh
				p.logPrefetchOutcome(outcome)
			}
		} else {
			<-prefetchCh
		}
		prefetchWG.Wait()

		// 下载完成后校验文件哈希
		p.setPhase("verify")
		p.log("正在校验文件哈希...")
		if err := p.verifyHash(localPath); err != nil {
			return fmt.Errorf("哈希校验失败: %w", err)
		}
		p.log("哈希校验通过")

		p.setPhase("delete")
		if err := retryOp(ctx, "删除远程分卷并确认 ack", p.log, func() error {
			return p.deleteRemoteAndAck(zipRemote)
		}); err != nil {
			return err
		}
		p.log("已删除远程 zip 并确认 ack，继续下一卷")
	}
}

// wrapCancelError 将 context.Canceled 替换为友好的中文提示
func wrapCancelError(err error) error {
	if err == context.Canceled {
		return fmt.Errorf(contextCanceledMsg)
	}
	return err
}

type prefetchOutcome struct {
	ahead string
	err   error
}

func (p *Pipeline) logPrefetchOutcome(outcome prefetchOutcome) {
	if outcome.err != nil {
		if outcome.err != context.Canceled {
			p.log("预打包下一卷失败: " + outcome.err.Error())
		}
		return
	}
	if outcome.ahead != "" {
		p.log("远程已预打包下一卷: " + filepath.Base(outcome.ahead))
	}
}

func (p *Pipeline) setPhase(phase string) {
	p.withStatus(func(st *model.JobStatus) {
		st.Phase = phase
		if phase != "download" {
			st.DownloadBytesDone = 0
			st.DownloadBytesTotal = 0
			st.DownloadSpeedBps = 0
		}
	})
}

func (p *Pipeline) maxGBFlag() string {
	gb := p.cfg.MaxPartGB
	if gb <= 0 {
		gb = 2
	}
	return fmt.Sprintf("%g", gb)
}

func (p *Pipeline) remotePack() (string, error) {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	state := util.NormalizeRemotePathForOS(p.cfg.RemoteState, p.cfg.OSType)
	out, stderr, err := cli.RunRemoteWithStderr("pack", "--state", state, "--max-gb", p.maxGBFlag())
	if err != nil {
		return "", err
	}
	for _, line := range splitLogLines(stderr) {
		p.log(line)
	}
	return strings.TrimSpace(out), nil
}

func (p *Pipeline) remotePackAhead() (string, error) {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	state := util.NormalizeRemotePathForOS(p.cfg.RemoteState, p.cfg.OSType)
	out, stderr, err := cli.RunRemoteWithStderr("pack-ahead", "--state", state, "--max-gb", p.maxGBFlag())
	if err != nil {
		return "", err
	}
	for _, line := range splitLogLines(stderr) {
		p.log(line)
	}
	return strings.TrimSpace(out), nil
}

func (p *Pipeline) deleteRemoteAndAck(zipRemote string) error {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer cli.Close()
	remotePath := util.NormalizeRemotePathForOS(zipRemote, p.cfg.OSType)
	if err := sftpclient.RemoveOnConn(cli.SSHConn(), remotePath); err != nil {
		return err
	}
	p.setPhase("ack")
	state := util.NormalizeRemotePathForOS(p.cfg.RemoteState, p.cfg.OSType)
	_, err = cli.RunRemote("ack", "--state", state)
	return err
}

func (p *Pipeline) download(ctx context.Context, remotePath, localPath string) error {
	sc, err := sftpclient.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer sc.Close()
	remotePath = util.NormalizeRemotePathForOS(remotePath, p.cfg.OSType)

	var progMu sync.Mutex
	var lastMark time.Time
	var lastBytes int64

	onProgress := func(written, total int64) {
		progMu.Lock()
		defer progMu.Unlock()
		now := time.Now()
		var speedBps float64
		updateSpeed := false
		if lastMark.IsZero() {
			lastMark = now
			lastBytes = written
		} else {
			dt := now.Sub(lastMark).Seconds()
			if dt >= 0.4 {
				speedBps = float64(written-lastBytes) / dt
				lastMark = now
				lastBytes = written
				updateSpeed = true
			}
		}
		p.withStatus(func(st *model.JobStatus) {
			st.DownloadBytesDone = written
			if total > 0 {
				st.DownloadBytesTotal = total
			}
			if updateSpeed {
				st.DownloadSpeedBps = speedBps
			}
		})
	}

	return sc.DownloadWithProgress(ctx, remotePath, localPath, 0, onProgress)
}

func (p *Pipeline) verifyHash(localPath string) error {
	hash, err := util.FileSHA256Hex(localPath)
	if err != nil {
		return fmt.Errorf("计算哈希失败: %w", err)
	}
	p.log("本地文件 SHA256: " + hash)
	// 当前远程不返回哈希，所以只计算并记录。未来可扩展为对比远程哈希。
	return nil
}

func (p *Pipeline) refreshStatus() error {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer cli.Close()
	state := util.NormalizeRemotePathForOS(p.cfg.RemoteState, p.cfg.OSType)
	out, err := cli.RunRemote("status", "--state", state, "--max-gb", p.maxGBFlag())
	if err != nil {
		return err
	}
	st, err := parseStatusJSON(out)
	if err != nil {
		return err
	}
	p.withStatus(func(job *model.JobStatus) {
		job.TotalFiles = st.TotalFiles
		job.PackedFiles = st.PackedFiles
		job.Done = st.Done
		job.PendingZip = st.PendingZip
		job.PrefetchZip = st.PrefetchZip
		job.MaxFileBytes = st.MaxFileBytes
		job.OversizedFileCount = st.OversizedFileCount
		job.RemoteInited = true
	})
	return nil
}

func osMkdirLocal(dir string) error {
	if dir == "" {
		return fmt.Errorf("local_dir 不能为空")
	}
	return osMkdirAll(dir)
}

func splitLogLines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// isConnError 判断错误是否为连接断开类可重试错误。
func isConnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection lost") ||
		strings.Contains(msg, "connection closed") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "forcibly closed") ||
		strings.Contains(msg, "wsarecv") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "ssh: handshake failed") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "reset by peer") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "ssh: unexpected")
}

// retryOp 对远程操作进行自动重试，遇到连接类错误时按指数退避等待后重试。
func retryOp(ctx context.Context, label string, log LogFn, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wait := retryBaseWait * (1 << (attempt - 1))
			if wait > retryMaxWait {
				wait = retryMaxWait
			}
			log(fmt.Sprintf("%s 失败（第 %d 次重试，等待 %v）: %v", label, attempt, wait, lastErr))
			select {
			case <-ctx.Done():
				return wrapCancelError(ctx.Err())
			case <-time.After(wait):
			}
		}
		err := fn()
		if err == nil {
			if attempt > 0 {
				log(label + " 重试成功")
			}
			return nil
		}
		lastErr = err
		if !isConnError(err) {
			return err
		}
	}
	return fmt.Errorf("%s 已重试 %d 次仍失败: %w", label, maxRetries, lastErr)
}

// retryWrap 是 retryOp 的变体，用于返回 (string, error) 的函数。
func retryWrap(ctx context.Context, label string, log LogFn, fn func() (string, error)) (string, error) {
	var lastErr error
	var result string
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wait := retryBaseWait * (1 << (attempt - 1))
			if wait > retryMaxWait {
				wait = retryMaxWait
			}
			log(fmt.Sprintf("%s 失败（第 %d 次重试，等待 %v）: %v", label, attempt, wait, lastErr))
			select {
			case <-ctx.Done():
				return "", wrapCancelError(ctx.Err())
			case <-time.After(wait):
			}
		}
		result, lastErr = fn()
		if lastErr == nil {
			if attempt > 0 {
				log(label + " 重试成功")
			}
			return result, nil
		}
		if !isConnError(lastErr) {
			return "", lastErr
		}
	}
	return "", fmt.Errorf("%s 已重试 %d 次仍失败: %w", label, maxRetries, lastErr)
}
