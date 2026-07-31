package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"z-server-backup-tools/backend/config"
	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/notify"
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

func (s *BackupService) OpenInExplorer(path string) error {
	return openPathInExplorer(path)
}

func (s *BackupService) TestEmailNotification(cfg model.BackupConfig) error {
	return notify.SendTestEmail(cfg)
}

func (s *BackupService) storedConfig() model.BackupConfig {
	if s.store == nil {
		return model.BackupConfig{Port: 22, MaxPartGB: 2}
	}
	return s.store.GetBackupConfig()
}

func (s *BackupService) SaveConnectionConfig(_ model.BackupConfig) error {
	return fmt.Errorf("请在「服务器管理」中保存 SSH 连接")
}

func (s *BackupService) SavePathsConfig(cfg model.BackupConfig) error {
	// 兼容旧前端：路径/SSH 已迁至服务器；此处仅保存邮件通知。
	return s.SaveNotifyConfig(cfg)
}

func (s *BackupService) SaveNotifyConfig(cfg model.BackupConfig) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	stored := s.storedConfig()
	stored.NotifyEmail = strings.TrimSpace(cfg.NotifyEmail)
	stored.SmtpHost = strings.TrimSpace(cfg.SmtpHost)
	stored.SmtpPort = cfg.SmtpPort
	stored.SmtpUser = strings.TrimSpace(cfg.SmtpUser)
	stored.SmtpPassword = cfg.SmtpPassword
	// 剥离全局 SSH/远程应用，避免 UI 误用残留连接字段
	stored.Host, stored.User, stored.Password, stored.RemoteAppDir = "", "", "", ""
	stored.Port = 22
	stored.MaxPartGB = 0
	stored.RemoteSrv = ""
	stored.RemoteState = ""
	stored.RemoteStaging = ""
	return s.store.SetBackupConfig(stored)
}

func (s *BackupService) GetServers() []model.Server {
	if s.store == nil {
		return nil
	}
	return s.store.GetServers()
}

func (s *BackupService) LookupServer(id string) (model.Server, error) {
	return lookupServer(s.store, id)
}

func (s *BackupService) SaveServer(srv model.Server) (model.Server, error) {
	if s.store == nil {
		return model.Server{}, fmt.Errorf("配置存储不可用")
	}
	normalized, err := normalizeServer(srv)
	if err != nil {
		return model.Server{}, err
	}
	servers := s.store.GetServers()
	found := false
	for i, existing := range servers {
		if existing.ID == normalized.ID {
			servers[i] = normalized
			found = true
			break
		}
	}
	if !found {
		servers = append(servers, normalized)
	}
	if err := s.store.SetServers(servers); err != nil {
		return model.Server{}, err
	}
	return normalized, nil
}

func (s *BackupService) DeleteServer(id string) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	return s.store.DeleteServer(strings.TrimSpace(id))
}

func (s *BackupService) BuildJobConfig(taskID string) (model.BackupConfig, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.BackupConfig{}, fmt.Errorf("请先选择备份任务")
	}
	for _, t := range s.GetTasks() {
		if t.ID == taskID {
			return s.resolveJobFromTask(t)
		}
	}
	return model.BackupConfig{}, fmt.Errorf("任务不存在: %s", taskID)
}

func (s *BackupService) resolveJobFromTask(task model.BackupTask) (model.BackupConfig, error) {
	srv, err := lookupServer(s.store, task.ServerID)
	if err != nil {
		return model.BackupConfig{}, err
	}
	if !srv.SupportMultiFile {
		return model.BackupConfig{}, fmt.Errorf("服务器不支持多文件备份")
	}
	return mergeServerTaskNotify(s.storedConfig(), srv, task), nil
}

// prepareMultiFileJob resolves SSH/remote_app/paths from store via task.server_id
// and ignores frontend-supplied connection fields. Keeps prepareBackupJob validation.
func (s *BackupService) prepareMultiFileJob(cfg model.BackupConfig) (model.BackupConfig, error) {
	merged, err := s.BuildJobConfig(cfg.TaskID)
	if err != nil {
		return model.BackupConfig{}, err
	}
	return prepareBackupJob(merged)
}

func (s *BackupService) GetTasks() []model.BackupTask {
	if s.store == nil {
		return nil
	}
	return s.store.GetBackupTasks()
}

func (s *BackupService) SaveTasks(tasks []model.BackupTask) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	normalized := make([]model.BackupTask, 0, len(tasks))
	seen := make(map[string]struct{}, len(tasks))
	for _, t := range tasks {
		t.ID = strings.TrimSpace(t.ID)
		t.Name = strings.TrimSpace(t.Name)
		t.ServerID = strings.TrimSpace(t.ServerID)
		t.RemoteSource = strings.TrimSpace(t.RemoteSource)
		t.LocalDir = strings.TrimSpace(t.LocalDir)
		t.PartNamePrefix = zipbak.SanitizePartPrefix(t.PartNamePrefix)
		if t.ID == "" {
			return fmt.Errorf("任务 ID 不能为空")
		}
		if t.ServerID == "" {
			return fmt.Errorf("任务 %s 未选择服务器", t.DisplayName())
		}
		srv, err := lookupServer(s.store, t.ServerID)
		if err != nil {
			return err
		}
		if !srv.SupportMultiFile {
			return fmt.Errorf("任务 %s 所选服务器不支持多文件备份", t.DisplayName())
		}
		if t.RemoteSource == "" {
			return fmt.Errorf("任务 %s 的远程源目录不能为空", t.DisplayName())
		}
		if t.LocalDir == "" {
			return fmt.Errorf("任务 %s 的本机保存目录不能为空", t.DisplayName())
		}
		if _, ok := seen[t.ID]; ok {
			return fmt.Errorf("任务 ID 重复: %s", t.ID)
		}
		seen[t.ID] = struct{}{}
		normalized = append(normalized, t)
	}
	return s.store.SetBackupTasks(normalized)
}

func (s *BackupService) GetActiveTaskID() string {
	if s.store == nil {
		return ""
	}
	return s.store.GetActiveTaskID()
}

func (s *BackupService) SetActiveTaskID(taskID string) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return s.store.SetActiveTaskID("")
	}
	for _, t := range s.store.GetBackupTasks() {
		if t.ID == taskID {
			return s.store.SetActiveTaskID(taskID)
		}
	}
	return fmt.Errorf("任务不存在: %s", taskID)
}

func (s *BackupService) activeTaskMerged() model.BackupConfig {
	base := s.storedConfig()
	activeID := s.store.GetActiveTaskID()
	if activeID == "" {
		return base.Resolved()
	}
	for _, t := range s.store.GetBackupTasks() {
		if t.ID == activeID {
			return t.MergeInto(base).Resolved()
		}
	}
	return base.Resolved()
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
	prepared, err := s.prepareMultiFileJob(cfg)
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
	prefix := zipbak.SanitizePartPrefix(prepared.PartNamePrefix)
	args := []string{
		"init",
		"--dir", source,
		"--state", state,
		"--staging", staging,
	}
	if prefix != "" {
		args = append(args, "--prefix", prefix)
	}
	_, err = cli.RunRemote(args...)
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
	return nil
}

func (s *BackupService) ResetBackupTask(cfg model.BackupConfig) error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("备份运行中，请先停止")
	}
	s.mu.Unlock()

	prepared, err := s.prepareMultiFileJob(cfg)
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
	s.clearBackupTiming()
	s.appendLog("任务已重置，下次备份将从第一个文件开始")
	return nil
}

func (s *BackupService) RefreshRemoteStatus(cfg model.BackupConfig) error {
	prepared, err := s.prepareMultiFileJob(cfg)
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
	if st.Done {
		s.mu.Unlock()
		s.clearBackupTiming()
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *BackupService) GetJobStatus() model.JobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	s.applyTimingToStatus(&st)
	return st
}

func (s *BackupService) GetLogs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *BackupService) StartBackup(cfg model.BackupConfig) error {
	prepared, err := s.prepareMultiFileJob(cfg)
	if err != nil {
		return err
	}
	if s.store != nil && strings.TrimSpace(prepared.TaskID) != "" {
		_ = s.store.SetActiveTaskID(prepared.TaskID)
	}

	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("备份已在运行")
	}
	s.mu.Unlock()

	if err := defaultJobGate.TryAcquireMulti(); err != nil {
		return err
	}

	s.mu.Lock()
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
		defaultJobGate.ReleaseMulti()
		s.mu.Lock()
		s.status.Running = false
		s.cancel = nil
		s.mu.Unlock()
	}()

	if st, err := s.queryRemoteStatusLocked(cfg); err == nil {
		s.mu.Lock()
		s.applyRemoteStatus(st, false)
		packed := s.status.PackedFiles
		s.mu.Unlock()
		s.beginBackupTiming(packed)
	} else {
		s.mu.Lock()
		packed := s.status.PackedFiles
		s.mu.Unlock()
		s.beginBackupTiming(packed)
	}

	pipe := zipbak.NewPipeline(cfg, s.appendLog, &s.status)
	s.appendLog("开始备份流水线")
	if err := pipe.Run(ctx); err != nil {
		s.mu.Lock()
		s.status.LastError = err.Error()
		st := s.status
		s.mu.Unlock()
		errMsg := err.Error()
		if strings.Contains(errMsg, "任务已取消") {
			s.appendLog("备份任务已取消")
		} else {
			s.appendLog("错误: " + errMsg)
			s.sendBackupNotification(cfg, notify.BackupResult{
				Success:      false,
				Host:         cfg.Host,
				RemoteSource: cfg.RemoteSource,
				LocalDir:     cfg.LocalDir,
				TotalFiles:   st.TotalFiles,
				PackedFiles:  st.PackedFiles,
				Error:        errMsg,
			})
		}
		s.app.Event.Emit("backup-error", errMsg)
		return
	}
	s.clearBackupTiming()
	s.mu.Lock()
	st := s.status
	s.mu.Unlock()
	s.sendBackupNotification(cfg, notify.BackupResult{
		Success:      true,
		Host:         cfg.Host,
		RemoteSource: cfg.RemoteSource,
		LocalDir:     cfg.LocalDir,
		TotalFiles:   st.TotalFiles,
		PackedFiles:  st.PackedFiles,
	})
	s.app.Event.Emit("backup-done", "ok")
}

func (s *BackupService) sendBackupNotification(cfg model.BackupConfig, result notify.BackupResult) {
	if strings.TrimSpace(cfg.NotifyEmail) == "" {
		return
	}
	if err := notify.SendBackupNotification(cfg, result); err != nil {
		s.appendLog("通知邮件发送失败: " + err.Error())
		return
	}
	if result.Success {
		s.appendLog("已发送备份完成通知邮件至 " + cfg.NotifyEmail)
	} else {
		s.appendLog("已发送备份异常通知邮件至 " + cfg.NotifyEmail)
	}
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
