package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"z-server-backup-tools/backend/config"
	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/remote"
	"z-server-backup-tools/backend/sftpclient"
	"z-server-backup-tools/backend/util"
	"z-server-backup-tools/backend/zipbak"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type BackupService struct {
	app           *application.App
	store         *config.Store
	mu            sync.Mutex
	remoteQueryMu sync.Mutex
	cancel        context.CancelFunc
	status        model.JobStatus
	logs          []string
}

func NewBackupService(app *application.App) *BackupService {
	store, _ := config.Default()
	return &BackupService{app: app, store: store}
}

func (s *BackupService) GetConfig() model.BackupConfig {
	if s.store == nil {
		return model.BackupConfig{Port: 22, MaxPartGB: 2}.Resolved()
	}
	return s.store.GetBackupConfig().Resolved()
}

func (s *BackupService) SaveConfig(cfg model.BackupConfig) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	if cfg.Host == "" || cfg.User == "" {
		return fmt.Errorf("主机与用户名不能为空")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.MaxPartGB <= 0 {
		cfg.MaxPartGB = 2
	}
	cfg.RemoteAppDir = strings.TrimSpace(cfg.RemoteAppDir)
	cfg.RemoteSource = strings.TrimSpace(cfg.RemoteSource)
	cfg.LocalDir = strings.TrimSpace(cfg.LocalDir)
	if cfg.RemoteAppDir == "" {
		cfg.RemoteAppDir = "D:/Tools/zipbak"
	}
	cfg.RemoteSrv = ""
	cfg.RemoteState = ""
	cfg.RemoteStaging = ""
	return s.store.SetBackupConfig(cfg)
}

func (s *BackupService) GetConfigPath() string {
	if s.store == nil {
		return ""
	}
	return s.store.ConfigPath()
}

func (s *BackupService) storedConfig() model.BackupConfig {
	if s.store == nil {
		return model.BackupConfig{Port: 22, MaxPartGB: 2}
	}
	return s.store.GetBackupConfig()
}

func (s *BackupService) SaveConnectionConfig(cfg model.BackupConfig) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	if cfg.Host == "" || cfg.User == "" {
		return fmt.Errorf("主机与用户名不能为空")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	stored := s.storedConfig()
	stored.Host = cfg.Host
	stored.User = cfg.User
	stored.Password = cfg.Password
	stored.Port = cfg.Port
	return s.store.SetBackupConfig(stored)
}

func (s *BackupService) SavePathsConfig(cfg model.BackupConfig) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	prev := s.storedConfig()
	oldMaxPartGB := prev.MaxPartGB
	stored := prev
	stored.RemoteAppDir = strings.TrimSpace(cfg.RemoteAppDir)
	stored.RemoteSource = strings.TrimSpace(cfg.RemoteSource)
	stored.LocalDir = strings.TrimSpace(cfg.LocalDir)
	if cfg.MaxPartGB > 0 {
		stored.MaxPartGB = cfg.MaxPartGB
	}
	if stored.MaxPartGB <= 0 {
		stored.MaxPartGB = 2
	}
	if stored.RemoteAppDir == "" {
		return fmt.Errorf("远程应用目录不能为空")
	}
	if stored.RemoteSource == "" {
		return fmt.Errorf("远程源目录不能为空")
	}
	if stored.LocalDir == "" {
		return fmt.Errorf("本机保存目录不能为空")
	}
	stored.RemoteSrv = ""
	stored.RemoteState = ""
	stored.RemoteStaging = ""
	resolved := stored.Resolved()
	if err := s.store.SetBackupConfig(resolved); err != nil {
		return err
	}
	if maxPartGBChanged(stored.MaxPartGB, oldMaxPartGB) {
		st, err := s.queryRemoteStatusLocked(resolved)
		if err == nil {
			s.mu.Lock()
			s.applyRemoteStatus(st, false)
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *BackupService) ListRemoteDirectories(cfg model.BackupConfig, pathHint string) (model.RemoteDirListing, error) {
	out := model.RemoteDirListing{}
	prepared, err := prepareSSH(cfg)
	if err != nil {
		return out, err
	}
	cli, err := sftpclient.Dial(prepared)
	if err != nil {
		return out, err
	}
	defer cli.Close()

	current, parent, names, err := cli.ListDirectories(pathHint)
	if err != nil {
		return out, err
	}
	entries := make([]model.RemoteDirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, model.RemoteDirEntry{
			Name: name,
			Path: sftpclient.JoinRemoteDirPublic(current, name),
		})
	}
	out.CurrentPath = current
	out.ParentPath = parent
	out.Entries = entries
	return out, nil
}

func (s *BackupService) TestConnection(cfg model.BackupConfig) error {
	prepared, err := prepareSSH(cfg)
	if err != nil {
		return err
	}
	cli, err := remote.Dial(prepared)
	if err != nil {
		return err
	}
	return cli.Close()
}

func (s *BackupService) InitRemote(cfg model.BackupConfig) error {
	prepared, err := prepareBackupJob(cfg)
	if err != nil {
		return err
	}
	cli, err := remote.Dial(prepared)
	if err != nil {
		return err
	}
	defer cli.Close()
	source := util.NormalizeRemotePath(prepared.RemoteSource)
	state := util.NormalizeRemotePath(prepared.RemoteState)
	staging := util.NormalizeRemotePath(prepared.RemoteStaging)
	_, err = cli.RunRemote(
		"init",
		"--dir", source,
		"--state", state,
		"--staging", staging,
	)
	if err != nil {
		return err
	}

	st, err := s.queryRemoteStatusLocked(prepared)
	if err != nil {
		s.mu.Lock()
		if isRemoteStateNotReady(err) {
			s.status.RemoteInited = false
			s.status.RemoteHint = remoteHintFromError(err)
		}
		s.mu.Unlock()
		s.appendLog("init 完成，但读取远程状态失败: " + err.Error())
		return nil
	}
	s.mu.Lock()
	s.applyRemoteStatus(st, true)
	s.status.RemoteHint = ""
	s.mu.Unlock()
	s.appendLog(fmt.Sprintf("远程 init 完成：清单共 %d 个文件，已打包 %d 个", st.TotalFiles, st.PackedFiles))
	if st.PendingZip != "" {
		s.appendLog("待处理分卷: " + st.PendingZip)
	}
	appendOversizedWarnings(s.appendLog, prepared)
	return nil
}

func (s *BackupService) ResetBackupTask(cfg model.BackupConfig) error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("备份运行中，请先停止")
	}
	s.mu.Unlock()

	prepared, err := prepareBackupJob(cfg)
	if err != nil {
		return err
	}
	cli, err := remote.Dial(prepared)
	if err != nil {
		return err
	}
	defer cli.Close()
	state := util.NormalizeRemotePath(prepared.RemoteState)
	if _, err := cli.RunRemote("reset", "--state", state); err != nil {
		return err
	}
	st, err := s.queryRemoteStatusLocked(prepared)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.applyRemoteStatus(st, true)
	s.status.Running = false
	s.status.Phase = ""
	s.status.CurrentPart = ""
	s.status.LocalFile = ""
	s.status.RemoteHint = ""
	s.logs = nil
	s.mu.Unlock()
	s.appendLog("任务已重置，下次备份将从第一个文件开始")
	return nil
}

func (s *BackupService) RefreshRemoteStatus(cfg model.BackupConfig) error {
	prepared, err := prepareBackupJob(cfg)
	if err != nil {
		s.mu.Lock()
		s.status.RemoteHint = ""
		s.mu.Unlock()
		return nil
	}
	st, err := s.queryRemoteStatusLocked(prepared)
	if err != nil {
		s.mu.Lock()
		s.status.RemoteInited = false
		s.status.RemoteHint = remoteHintFromError(err)
		if isRemoteStateNotReady(err) {
			s.status.TotalFiles = 0
			s.status.PackedFiles = 0
			s.status.PendingZip = ""
			s.status.Done = false
			s.status.MaxFileBytes = 0
			s.status.OversizedFileCount = 0
		}
		s.mu.Unlock()
		return nil
	}
	s.mu.Lock()
	s.applyRemoteStatus(st, false)
	s.status.RemoteHint = ""
	s.mu.Unlock()
	return nil
}

func (s *BackupService) GetJobStatus() model.JobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *BackupService) GetLogs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *BackupService) StartBackup(cfg model.BackupConfig) error {
	prepared, err := prepareBackupJob(cfg)
	if err != nil {
		return err
	}
	if s.store != nil {
		_ = s.store.SetBackupConfig(prepared)
	}

	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("备份已在运行")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.status = model.JobStatus{Running: true, Phase: "starting"}
	s.logs = nil
	jobCfg := prepared
	s.mu.Unlock()

	go s.runPipeline(ctx, jobCfg)
	return nil
}

func (s *BackupService) StopBackup() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *BackupService) runPipeline(ctx context.Context, cfg model.BackupConfig) {
	defer func() {
		s.mu.Lock()
		s.status.Running = false
		s.cancel = nil
		s.mu.Unlock()
	}()

	pipe := zipbak.NewPipeline(cfg, s.appendLog, &s.status)
	s.appendLog("开始备份流水线")
	if err := pipe.Run(ctx); err != nil {
		s.mu.Lock()
		s.status.LastError = err.Error()
		s.mu.Unlock()
		errMsg := err.Error()
		if strings.Contains(errMsg, "任务已取消") {
			s.appendLog("备份任务已取消")
		} else {
			s.appendLog("错误: " + errMsg)
			s.appendLog("可再次点击「开始备份」从远程断点继续（有未传完的分卷会先重试该卷；勿点「重置任务」）。")
		}
		s.app.Event.Emit("backup-error", errMsg)
		return
	}
	s.app.Event.Emit("backup-done", "ok")
}

func (s *BackupService) appendLog(line string) {
	s.mu.Lock()
	s.logs = append(s.logs, line)
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
	s.mu.Unlock()
	s.app.Event.Emit("backup-log", line)
}
