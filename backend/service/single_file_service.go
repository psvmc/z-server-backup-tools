package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"z-server-backup-tools/backend/config"
	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/notify"
	"z-server-backup-tools/backend/sftpclient"
	"z-server-backup-tools/backend/util"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type SingleFileBackupService struct {
	app    *application.App
	store  *config.Store
	mu     sync.Mutex
	cancel context.CancelFunc
	status model.JobStatus
	logs   []string
}

func NewSingleFileBackupService(app *application.App) *SingleFileBackupService {
	store, _ := config.Default()
	return &SingleFileBackupService{app: app, store: store}
}

func singleFileLocalPath(remoteFile, localDir string) (string, error) {
	remoteFile = strings.TrimSpace(remoteFile)
	localDir = strings.TrimSpace(localDir)
	if remoteFile == "" {
		return "", fmt.Errorf("远程文件路径不能为空")
	}
	if localDir == "" {
		return "", fmt.Errorf("本机保存目录不能为空")
	}
	normalized := strings.ReplaceAll(remoteFile, `\`, `/`)
	base := filepath.Base(filepath.FromSlash(normalized))
	if base == "." || base == string(filepath.Separator) || base == "/" || base == `\` {
		return "", fmt.Errorf("无法从远程路径解析文件名")
	}
	return filepath.Join(localDir, base), nil
}

func (s *SingleFileBackupService) GetConfig() model.SingleFileConfig {
	if s.store == nil {
		return model.SingleFileConfig{}
	}
	return s.store.GetSingleFileConfig()
}

func (s *SingleFileBackupService) SaveConfig(cfg model.SingleFileConfig) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	cfg.RemoteFile = strings.TrimSpace(cfg.RemoteFile)
	cfg.LocalDir = strings.TrimSpace(cfg.LocalDir)
	if cfg.RemoteFile == "" {
		return fmt.Errorf("远程文件路径不能为空")
	}
	if cfg.LocalDir == "" {
		return fmt.Errorf("本机保存目录不能为空")
	}
	return s.store.SetSingleFileConfig(cfg)
}

func (s *SingleFileBackupService) GetStatus() model.JobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *SingleFileBackupService) GetLogs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *SingleFileBackupService) StartDownload(sshCfg model.BackupConfig, paths model.SingleFileConfig) error {
	prepared, err := prepareSSH(sshCfg)
	if err != nil {
		return err
	}
	paths.RemoteFile = strings.TrimSpace(paths.RemoteFile)
	paths.LocalDir = strings.TrimSpace(paths.LocalDir)
	localPath, err := singleFileLocalPath(paths.RemoteFile, paths.LocalDir)
	if err != nil {
		return err
	}

	if err := defaultJobGate.TryAcquireSingle(); err != nil {
		return err
	}

	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		defaultJobGate.ReleaseSingle()
		return fmt.Errorf("单文件下载已在运行")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.status = model.JobStatus{
		Running:   true,
		Phase:     "download",
		LocalFile: localPath,
	}
	s.logs = nil
	jobCfg := prepared
	remoteFile := paths.RemoteFile
	s.mu.Unlock()

	go s.runDownload(ctx, jobCfg, remoteFile, localPath)
	return nil
}

func (s *SingleFileBackupService) StopDownload() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *SingleFileBackupService) runDownload(ctx context.Context, cfg model.BackupConfig, remoteFile, localPath string) {
	defer func() {
		defaultJobGate.ReleaseSingle()
		s.mu.Lock()
		s.status.Running = false
		s.cancel = nil
		s.mu.Unlock()
	}()

	remotePath := util.NormalizeRemotePath(remoteFile)
	s.appendLog(fmt.Sprintf("开始下载：%s → %s", remotePath, localPath))

	sc, err := sftpclient.Dial(cfg)
	if err != nil {
		s.failDownload(cfg, remoteFile, localPath, err)
		return
	}
	defer sc.Close()

	var progMu sync.Mutex
	var lastMark time.Time
	var lastBytes int64

	onProgress := func(written, total int64) {
		progMu.Lock()
		defer progMu.Unlock()
		s.mu.Lock()
		s.status.DownloadBytesDone = written
		if total > 0 {
			s.status.DownloadBytesTotal = total
		}
		now := time.Now()
		if lastMark.IsZero() {
			lastMark = now
			lastBytes = written
			s.mu.Unlock()
			return
		}
		dt := now.Sub(lastMark).Seconds()
		if dt >= 0.4 {
			s.status.DownloadSpeedBps = float64(written-lastBytes) / dt
			lastMark = now
			lastBytes = written
		}
		s.mu.Unlock()
	}

	if err := sc.DownloadWithProgress(ctx, remotePath, localPath, 0, onProgress); err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			s.appendLog("单文件下载已取消")
			s.app.Event.Emit("singlefile-error", "任务已取消")
			return
		}
		s.failDownload(cfg, remoteFile, localPath, err)
		return
	}

	s.mu.Lock()
	s.status.Phase = "done"
	s.status.Done = true
	s.mu.Unlock()
	s.appendLog("下载完成: " + localPath)
	s.sendNotification(cfg, true, remoteFile, localPath, "")
	s.app.Event.Emit("singlefile-done", localPath)
}

func (s *SingleFileBackupService) failDownload(cfg model.BackupConfig, remoteFile, localPath string, err error) {
	errMsg := err.Error()
	s.mu.Lock()
	s.status.LastError = errMsg
	s.mu.Unlock()
	s.appendLog("错误: " + errMsg)
	s.sendNotification(cfg, false, remoteFile, localPath, errMsg)
	s.app.Event.Emit("singlefile-error", errMsg)
}

func (s *SingleFileBackupService) sendNotification(cfg model.BackupConfig, success bool, remoteFile, localPath, errMsg string) {
	if strings.TrimSpace(cfg.NotifyEmail) == "" {
		return
	}
	if err := notify.SendSingleFileNotification(cfg, success, remoteFile, localPath, errMsg); err != nil {
		s.appendLog("通知邮件发送失败: " + err.Error())
		return
	}
	if success {
		s.appendLog("已发送单文件下载完成通知邮件至 " + cfg.NotifyEmail)
	} else {
		s.appendLog("已发送单文件下载异常通知邮件至 " + cfg.NotifyEmail)
	}
}

func (s *SingleFileBackupService) appendLog(line string) {
	s.mu.Lock()
	s.logs = append(s.logs, line)
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
	s.mu.Unlock()
	s.app.Event.Emit("singlefile-log", line)
}
