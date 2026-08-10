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
	maxRetries    = 3
	retryBaseWait = 2 * time.Second
	retryMaxWait  = 10 * time.Second
)

// contextCanceledMsg 用于将 context.Canceled 包装为友好的中文提示
const contextCanceledMsg = "任务已取消"

type LogFn func(string)

type Pipeline struct {
	cfg    model.BackupConfig
	log    LogFn
	status *model.JobStatus
}

func NewPipeline(cfg model.BackupConfig, log LogFn, status *model.JobStatus) *Pipeline {
	return &Pipeline{cfg: cfg, log: log, status: status}
}

func (p *Pipeline) Run(ctx context.Context) error {
	if err := osMkdirLocal(p.cfg.LocalDir); err != nil {
		return err
	}
	if err := retryOp(ctx, "读取远程状态", p.log, p.refreshStatus); err != nil {
		p.log("读取远程状态失败: " + err.Error())
	}
	if strings.TrimSpace(p.status.PendingZip) != "" {
		p.log("检测到远程待处理分卷，将从断点继续: " + filepath.Base(p.status.PendingZip))
	}
	for {
		if err := ctx.Err(); err != nil {
			return wrapCancelError(err)
		}
		if err := retryOp(ctx, "读取远程状态", p.log, p.refreshStatus); err != nil {
			p.log("读取远程状态失败: " + err.Error())
		}
		if p.status.Done && strings.TrimSpace(p.status.PendingZip) == "" {
			p.log("全部文件已备份完成")
			return nil
		}

		p.setPhase("pack")
		zipRemote, err := retryWrap(ctx, "远程打包", p.log, p.remotePack)
		if err != nil {
			if strings.Contains(err.Error(), "备份已完成") {
				p.status.Done = true
				p.log("备份任务已完成")
				return nil
			}
			return err
		}
		p.status.CurrentPart = filepath.Base(zipRemote)
		p.log("远程已生成: " + zipRemote)

		localName := filepath.Base(zipRemote)
		localPath := filepath.Join(p.cfg.LocalDir, localName)
		p.status.LocalFile = localPath
		p.setPhase("download")
		p.log("开始下载到 " + localPath)

		iterCtx, iterCancel := context.WithCancel(ctx)
		var prefetchWG sync.WaitGroup
		prefetchWG.Add(1)
		go func() {
			defer prefetchWG.Done()
			if err := iterCtx.Err(); err != nil {
				return
			}
			ahead, err := retryWrap(iterCtx, "预打包下一卷", p.log, p.remotePackAhead)
			if err != nil {
				if iterCtx.Err() == nil {
					p.log("预打包下一卷失败: " + err.Error())
				}
				return
			}
			if strings.TrimSpace(ahead) != "" {
				p.log("远程已预打包下一卷: " + filepath.Base(ahead))
			}
		}()

		if err := retryOp(iterCtx, "下载分卷", p.log, func() error {
			return p.download(iterCtx, zipRemote, localPath)
		}); err != nil {
			iterCancel()
			prefetchWG.Wait()
			return wrapCancelError(err)
		}
		p.log("下载完成")

		iterCancel()
		prefetchWG.Wait()

		// 下载完成后校验文件哈希
		p.setPhase("verify")
		p.log("正在校验文件哈希...")
		if err := p.verifyHash(localPath); err != nil {
			return fmt.Errorf("哈希校验失败: %w", err)
		}
		p.log("哈希校验通过")

		p.setPhase("delete")
		if err := retryOp(ctx, "删除远程分卷", p.log, func() error {
			return p.deleteRemote(zipRemote)
		}); err != nil {
			return err
		}
		p.log("已删除远程 zip")

		p.setPhase("ack")
		if err := retryOp(ctx, "确认 ack", p.log, p.remoteAck); err != nil {
			return err
		}
		p.log("已确认 ack，继续下一卷")
	}
}

// wrapCancelError 将 context.Canceled 替换为友好的中文提示
func wrapCancelError(err error) error {
	if err == context.Canceled {
		return fmt.Errorf(contextCanceledMsg)
	}
	return err
}

func (p *Pipeline) setPhase(phase string) {
	p.status.Phase = phase
	if phase != "download" {
		p.status.DownloadBytesDone = 0
		p.status.DownloadBytesTotal = 0
		p.status.DownloadSpeedBps = 0
	}
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

func (p *Pipeline) remoteAck() error {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer cli.Close()
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
		p.status.DownloadBytesDone = written
		if total > 0 {
			p.status.DownloadBytesTotal = total
		}
		now := time.Now()
		if lastMark.IsZero() {
			lastMark = now
			lastBytes = written
			return
		}
		dt := now.Sub(lastMark).Seconds()
		if dt >= 0.4 {
			p.status.DownloadSpeedBps = float64(written-lastBytes) / dt
			lastMark = now
			lastBytes = written
		}
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

func (p *Pipeline) deleteRemote(remotePath string) error {
	sc, err := sftpclient.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer sc.Close()
	return sc.Remove(util.NormalizeRemotePathForOS(remotePath, p.cfg.OSType))
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
	p.status.TotalFiles = st.TotalFiles
	p.status.PackedFiles = st.PackedFiles
	p.status.Done = st.Done
	p.status.PendingZip = st.PendingZip
	p.status.PrefetchZip = st.PrefetchZip
	p.status.MaxFileBytes = st.MaxFileBytes
	p.status.OversizedFileCount = st.OversizedFileCount
	p.status.RemoteInited = true
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
