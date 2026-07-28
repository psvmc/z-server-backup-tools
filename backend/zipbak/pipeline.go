package zipbak

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/remote"
	"z-server-backup-tools/backend/sftpclient"
	"z-server-backup-tools/backend/util"
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
	if err := p.refreshStatus(); err != nil {
		p.log("读取远程状态失败: " + err.Error())
	}
	p.logOversizedWarnings()
	if strings.TrimSpace(p.status.PendingZip) != "" {
		p.log("检测到远程待处理分卷，将从断点继续: " + filepath.Base(p.status.PendingZip))
	}
	for {
		if err := ctx.Err(); err != nil {
			return wrapCancelError(err)
		}
		if err := p.refreshStatus(); err != nil {
			p.log("读取远程状态失败: " + err.Error())
		}
		if p.status.Done && strings.TrimSpace(p.status.PendingZip) == "" {
			p.log("全部文件已备份完成")
			return nil
		}

		p.setPhase("pack")
		zipRemote, err := p.remotePack()
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

		var prefetchWG sync.WaitGroup
		prefetchWG.Add(1)
		go func() {
			defer prefetchWG.Done()
			if err := ctx.Err(); err != nil {
				return
			}
			ahead, err := p.remotePackAhead()
			if err != nil {
				p.log("预打包下一卷失败: " + err.Error())
				return
			}
			if strings.TrimSpace(ahead) != "" {
				p.log("远程已预打包下一卷: " + filepath.Base(ahead))
			}
		}()

		if err := p.download(ctx, zipRemote, localPath); err != nil {
			return wrapCancelError(err)
		}
		p.log("下载完成")

		prefetchDone := make(chan struct{})
		go func() {
			prefetchWG.Wait()
			close(prefetchDone)
		}()
		select {
		case <-ctx.Done():
			return wrapCancelError(ctx.Err())
		case <-prefetchDone:
		}

		// 下载完成后校验文件哈希
		p.setPhase("verify")
		p.log("正在校验文件哈希...")
		if err := p.verifyHash(localPath); err != nil {
			return fmt.Errorf("哈希校验失败: %w", err)
		}
		p.log("哈希校验通过")

		p.setPhase("delete")
		if err := p.deleteRemote(zipRemote); err != nil {
			return err
		}
		p.log("已删除远程 zip")

		p.setPhase("ack")
		if err := p.remoteAck(); err != nil {
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
	state := util.NormalizeRemotePath(p.cfg.RemoteState)
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
	state := util.NormalizeRemotePath(p.cfg.RemoteState)
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
	state := util.NormalizeRemotePath(p.cfg.RemoteState)
	_, err = cli.RunRemote("ack", "--state", state)
	return err
}

func (p *Pipeline) download(ctx context.Context, remotePath, localPath string) error {
	sc, err := sftpclient.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer sc.Close()
	remotePath = util.NormalizeRemotePath(remotePath)
	return sc.Download(ctx, remotePath, localPath)
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
	return sc.Remove(util.NormalizeRemotePath(remotePath))
}

func (p *Pipeline) refreshStatus() error {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return err
	}
	defer cli.Close()
	state := util.NormalizeRemotePath(p.cfg.RemoteState)
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

func (p *Pipeline) logOversizedWarnings() {
	cli, err := remote.Dial(p.cfg)
	if err != nil {
		return
	}
	defer cli.Close()
	state := util.NormalizeRemotePath(p.cfg.RemoteState)
	out, err := cli.RunRemote("oversized", "--state", state, "--max-gb", p.maxGBFlag())
	if err != nil {
		return
	}
	items, err := ParseOversizedJSON(out)
	if err != nil {
		return
	}
	for _, line := range OversizedWarningLines(p.cfg.MaxPartBytes(), items) {
		p.log(line)
	}
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
