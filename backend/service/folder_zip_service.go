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
	"z-server-backup-tools/backend/remote"
	"z-server-backup-tools/backend/sftpclient"
	"z-server-backup-tools/backend/util"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type FolderZipBackupService struct {
	app          *application.App
	store        *config.Store
	mu           sync.Mutex
	cancel       context.CancelFunc
	status       model.JobStatus
	logs         []string
	pendingQueue []string
	queueTotal   int
}

func NewFolderZipBackupService(app *application.App) *FolderZipBackupService {
	store, _ := config.Default()
	return &FolderZipBackupService{app: app, store: store}
}

func taskToFolderZipPaths(t model.BackupTask) (model.FolderZipConfig, error) {
	if t.NormalizedKind() != "folder_zip" {
		return model.FolderZipConfig{}, fmt.Errorf("不是文件夹压缩备份任务")
	}
	if strings.TrimSpace(t.ServerID) == "" {
		return model.FolderZipConfig{}, fmt.Errorf("请选择服务器")
	}
	rf := strings.TrimSpace(t.RemoteSource)
	ld := strings.TrimSpace(t.LocalDir)
	if rf == "" || ld == "" {
		return model.FolderZipConfig{}, fmt.Errorf("远程文件夹与本机目录不能为空")
	}
	return model.FolderZipConfig{
		ServerID:       t.ServerID,
		RemoteFolder:   rf,
		LocalDir:       ld,
		IgnorePatterns: util.NormalizeIgnorePatterns(t.IgnorePatterns),
	}, nil
}

func folderZipLocalPath(remoteFolder, localDir string) (string, error) {
	remoteFolder = strings.TrimSpace(remoteFolder)
	localDir = strings.TrimSpace(localDir)
	if remoteFolder == "" {
		return "", fmt.Errorf("远程文件夹路径不能为空")
	}
	if localDir == "" {
		return "", fmt.Errorf("本机保存目录不能为空")
	}
	normalized := strings.ReplaceAll(remoteFolder, `\`, `/`)
	normalized = strings.TrimSuffix(normalized, `/`)
	base := filepath.Base(filepath.FromSlash(normalized))
	if base == "." || base == string(filepath.Separator) || base == "/" || base == `\` {
		return "", fmt.Errorf("无法从远程路径解析文件夹名")
	}
	return filepath.Join(localDir, base+".zip"), nil
}

func resolveFolderZipTaskQueue(tasks []model.BackupTask, taskIDs []string) ([]string, error) {
	if len(taskIDs) == 0 {
		return nil, fmt.Errorf("请至少选择一个备份任务")
	}
	byID := make(map[string]model.BackupTask, len(tasks))
	for _, t := range tasks {
		if t.NormalizedKind() == "folder_zip" {
			byID[t.ID] = t
		}
	}
	queue := make([]string, 0, len(taskIDs))
	seen := make(map[string]bool, len(taskIDs))
	for _, rawID := range taskIDs {
		id := strings.TrimSpace(rawID)
		if id == "" || seen[id] {
			continue
		}
		t, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("任务无效或已删除")
		}
		if _, err := taskToFolderZipPaths(t); err != nil {
			label := strings.TrimSpace(t.Name)
			if label == "" {
				label = strings.TrimSpace(t.RemoteSource)
			}
			if label == "" {
				label = id
			}
			return nil, fmt.Errorf("任务「%s」配置不完整", label)
		}
		seen[id] = true
		queue = append(queue, id)
	}
	if len(queue) == 0 {
		return nil, fmt.Errorf("请至少选择一个有效的备份任务")
	}
	return queue, nil
}

func (s *FolderZipBackupService) findFolderZipTask(id string) (model.BackupTask, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.BackupTask{}, fmt.Errorf("任务 ID 不能为空")
	}
	for _, t := range s.store.GetBackupTasks() {
		if t.ID == id {
			if t.NormalizedKind() != "folder_zip" {
				return model.BackupTask{}, fmt.Errorf("不是文件夹压缩备份任务")
			}
			return t, nil
		}
	}
	return model.BackupTask{}, fmt.Errorf("文件夹压缩备份任务不存在")
}

func (s *FolderZipBackupService) prepareFolderZipJob(taskID string) (model.BackupConfig, string, string, []string, error) {
	task, err := s.findFolderZipTask(taskID)
	if err != nil {
		return model.BackupConfig{}, "", "", nil, err
	}
	paths, err := taskToFolderZipPaths(task)
	if err != nil {
		return model.BackupConfig{}, "", "", nil, err
	}
	srv, err := lookupServer(s.store, task.ServerID)
	if err != nil {
		return model.BackupConfig{}, "", "", nil, err
	}
	notifyCfg := s.store.GetBackupConfig()
	cfg := srv.ApplyTo(notifyCfg)
	prepared, err := prepareSSH(cfg)
	if err != nil {
		return model.BackupConfig{}, "", "", nil, err
	}
	localPath, err := folderZipLocalPath(paths.RemoteFolder, paths.LocalDir)
	if err != nil {
		return model.BackupConfig{}, "", "", nil, err
	}
	return prepared, paths.RemoteFolder, localPath, paths.IgnorePatterns, nil
}

func (s *FolderZipBackupService) clearQueueLocked() {
	s.pendingQueue = nil
	s.queueTotal = 0
	s.status.QueueIndex = 0
	s.status.QueueTotal = 0
}

func (s *FolderZipBackupService) finishQueueItem(continueQueue bool) (nextID string, hasNext bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if continueQueue && len(s.pendingQueue) > 0 {
		nextID = s.pendingQueue[0]
		s.pendingQueue = s.pendingQueue[1:]
		return nextID, true
	}
	s.clearQueueLocked()
	s.status.Running = false
	s.cancel = nil
	defaultJobGate.ReleaseFolderZip()
	return "", false
}

func (s *FolderZipBackupService) GetStatus() model.JobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *FolderZipBackupService) GetLogs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *FolderZipBackupService) StartBackup(taskIDs []string) error {
	if s.store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	queue, err := resolveFolderZipTaskQueue(s.store.GetBackupTasks(), taskIDs)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("文件夹压缩备份已在运行")
	}
	s.mu.Unlock()

	if err := defaultJobGate.TryAcquireFolderZip(); err != nil {
		return err
	}

	s.mu.Lock()
	s.pendingQueue = append([]string{}, queue[1:]...)
	s.queueTotal = len(queue)
	s.mu.Unlock()

	if err := s.startTask(queue[0], true); err != nil {
		s.mu.Lock()
		s.clearQueueLocked()
		s.status.Running = false
		s.mu.Unlock()
		defaultJobGate.ReleaseFolderZip()
		return err
	}
	return nil
}

func (s *FolderZipBackupService) startTask(taskID string, first bool) error {
	jobCfg, remoteFolder, localPath, ignorePatterns, err := s.prepareFolderZipJob(taskID)
	if err != nil {
		return err
	}
	if err := s.store.SetActiveFolderZipTaskID(taskID); err != nil {
		return err
	}

	s.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	nowMs := time.Now().UnixMilli()
	queueIndex := s.queueTotal - len(s.pendingQueue)
	queueTotal := s.queueTotal
	s.status = model.JobStatus{
		Running:           true,
		Phase:             "compress",
		LocalFile:         localPath,
		TimingStartedAtMs: nowMs,
		QueueIndex:        queueIndex,
		QueueTotal:        queueTotal,
	}
	if first {
		s.logs = nil
	}
	jobCfgCopy := jobCfg
	remoteFolderCopy := remoteFolder
	ignoreCopy := append([]string(nil), ignorePatterns...)
	s.mu.Unlock()

	if !first {
		s.appendLog("---")
	}
	if queueTotal > 1 {
		s.appendLog(fmt.Sprintf("队列任务 %d/%d", queueIndex, queueTotal))
	}
	go s.runBackup(ctx, jobCfgCopy, remoteFolderCopy, localPath, ignoreCopy)
	return nil
}

func (s *FolderZipBackupService) StopBackup() {
	s.mu.Lock()
	s.clearQueueLocked()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *FolderZipBackupService) runBackup(ctx context.Context, cfg model.BackupConfig, remoteFolder, localPath string, ignorePatterns []string) {
	continueQueue := false
	defer func() {
		nextID, hasNext := s.finishQueueItem(continueQueue)
	if hasNext {
		if err := s.startTask(nextID, false); err != nil {
			s.mu.Lock()
			s.clearQueueLocked()
			s.status.Running = false
			s.cancel = nil
			s.mu.Unlock()
			defaultJobGate.ReleaseFolderZip()
			s.appendLog("队列下一任务启动失败: " + err.Error())
			s.app.Event.Emit("folderzip-error", err.Error())
		}
	}
	}()

	remotePath := util.NormalizeRemotePathForOS(remoteFolder, cfg.OSType)
	startedAt := time.Now()
	s.appendLog(fmt.Sprintf("开始压缩备份：%s → %s", remotePath, localPath))
	if len(ignorePatterns) > 0 {
		s.appendLog(fmt.Sprintf("忽略规则 %d 条：%s", len(ignorePatterns), strings.Join(ignorePatterns, "；")))
	}

	onProgress := s.makeProgressCallback()

	serverOK := false
	if err := s.tryServerSideBackup(ctx, cfg, remotePath, localPath, ignorePatterns, onProgress); err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			elapsed := time.Since(startedAt)
			s.appendLog(fmt.Sprintf("文件夹压缩备份已取消（用时 %s）", util.FormatDuration(elapsed.Milliseconds())))
			s.app.Event.Emit("folderzip-error", "任务已取消")
			return
		}
		s.appendLog("服务端压缩不可用，回退为本机边下载边压缩: " + err.Error())
	} else {
		serverOK = true
	}
	if !serverOK {
		if err := s.runClientSideBackup(ctx, cfg, remotePath, localPath, ignorePatterns, onProgress); err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				elapsed := time.Since(startedAt)
				s.appendLog(fmt.Sprintf("文件夹压缩备份已取消（用时 %s）", util.FormatDuration(elapsed.Milliseconds())))
				s.app.Event.Emit("folderzip-error", "任务已取消")
				return
			}
			s.failBackup(cfg, remoteFolder, localPath, err, startedAt)
			return
		}
	}

	elapsed := time.Since(startedAt)
	s.mu.Lock()
	s.status.Phase = "done"
	s.status.Done = true
	s.status.TimingEstimatedTotalMs = elapsed.Milliseconds()
	s.mu.Unlock()
	s.appendLog(fmt.Sprintf("压缩备份完成: %s（用时 %s）", localPath, util.FormatDuration(elapsed.Milliseconds())))
	s.sendNotification(cfg, true, remoteFolder, localPath, "")
	s.app.Event.Emit("folderzip-done", localPath)
	continueQueue = true
}

func (s *FolderZipBackupService) makeProgressCallback() func(written, total int64) {
	var progMu sync.Mutex
	var lastMark time.Time
	var lastBytes int64
	return func(written, total int64) {
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
}

func (s *FolderZipBackupService) ignoreLogCallback() func(string) {
	return func(p string) {
		s.appendLog("忽略: " + p)
	}
}

func (s *FolderZipBackupService) tryServerSideBackup(
	ctx context.Context,
	cfg model.BackupConfig,
	remotePath, localPath string,
	ignorePatterns []string,
	onProgress func(written, total int64),
) error {
	rc, err := remote.Dial(cfg)
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := rc.EnsureZipTool(s.appendLog); err != nil {
		return err
	}

	zipBase := strings.TrimSuffix(filepath.Base(localPath), ".zip")
	token := newEntityID()
	remoteZip, err := rc.ResolveRemoteTempZipPath(zipBase, token)
	if err != nil {
		return err
	}
	s.appendLog("远程临时压缩包: " + remoteZip)

	var listFile string
	if len(ignorePatterns) > 0 {
		s.appendLog("正在扫描远程文件并应用忽略规则…")
		s.mu.Lock()
		s.status.Phase = "scan"
		s.mu.Unlock()

		sc, err := sftpclient.NewFromConn(rc.SSHConn())
		if err != nil {
			return err
		}
		paths, err := sc.ListIncludedRelPaths(remotePath, cfg.OSType, ignorePatterns, s.ignoreLogCallback())
		sc.Close()
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("过滤后没有可备份的文件")
		}
		s.appendLog(fmt.Sprintf("过滤后待压缩文件 %d 个", len(paths)))

		listFile, err = rc.ResolveRemoteTempListPath(token)
		if err != nil {
			return err
		}
		s.appendLog("远程文件列表: " + listFile)

		sc, err = sftpclient.NewFromConn(rc.SSHConn())
		if err != nil {
			return err
		}
		listContent := strings.Join(paths, "\n") + "\n"
		if err := sc.WriteRemoteFile(util.NormalizeRemotePathForOS(listFile, cfg.OSType), []byte(listContent)); err != nil {
			sc.Close()
			return fmt.Errorf("上传文件列表失败: %w", err)
		}
		sc.Close()
	}

	s.appendLog("开始在服务端压缩…")
	s.mu.Lock()
	s.status.Phase = "compress"
	s.mu.Unlock()

	compressErr := make(chan error, 1)
	go func() {
		if listFile != "" {
			compressErr <- rc.CompressFolderRemoteWithList(remotePath, remoteZip, listFile)
			return
		}
		compressErr <- rc.CompressFolderRemote(remotePath, remoteZip)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-compressErr:
		if err != nil {
			return fmt.Errorf("服务端压缩失败: %w", err)
		}
	}

	defer func() {
		if listFile != "" {
			if err := rc.RemoveRemoteFile(listFile); err != nil {
				s.appendLog("删除远程文件列表失败: " + err.Error())
			}
		}
		if err := rc.RemoveRemoteFile(remoteZip); err != nil {
			s.appendLog("删除远程临时压缩包失败: " + err.Error())
		} else {
			s.appendLog("已删除远程临时压缩包")
		}
	}()

	s.appendLog("服务端压缩完成，开始下载…")
	s.mu.Lock()
	s.status.Phase = "download"
	s.mu.Unlock()

	sc, err := sftpclient.NewFromConn(rc.SSHConn())
	if err != nil {
		return err
	}
	defer sc.Close()

	sftpZip := util.NormalizeRemotePathForOS(remoteZip, cfg.OSType)
	return sc.DownloadWithProgress(ctx, sftpZip, localPath, 0, onProgress)
}

func (s *FolderZipBackupService) runClientSideBackup(
	ctx context.Context,
	cfg model.BackupConfig,
	remotePath, localPath string,
	ignorePatterns []string,
	onProgress func(written, total int64),
) error {
	sc, err := sftpclient.Dial(cfg)
	if err != nil {
		return err
	}
	defer sc.Close()
	s.mu.Lock()
	s.status.Phase = "compress"
	s.mu.Unlock()
	return sc.ZipRemoteFolder(ctx, remotePath, cfg.OSType, localPath, ignorePatterns, s.ignoreLogCallback(), onProgress)
}

func (s *FolderZipBackupService) failBackup(cfg model.BackupConfig, remoteFolder, localPath string, err error, startedAt time.Time) {
	errMsg := err.Error()
	elapsed := time.Since(startedAt)
	s.mu.Lock()
	s.status.LastError = errMsg
	s.status.TimingEstimatedTotalMs = elapsed.Milliseconds()
	s.mu.Unlock()
	s.appendLog(fmt.Sprintf("错误: %s（用时 %s）", errMsg, util.FormatDuration(elapsed.Milliseconds())))
	s.sendNotification(cfg, false, remoteFolder, localPath, errMsg)
	s.app.Event.Emit("folderzip-error", errMsg)
}

func (s *FolderZipBackupService) sendNotification(cfg model.BackupConfig, success bool, remoteFolder, localPath, errMsg string) {
	if strings.TrimSpace(cfg.NotifyEmail) == "" {
		return
	}
	if err := notify.SendFolderZipNotification(cfg, success, remoteFolder, localPath, errMsg); err != nil {
		s.appendLog("通知邮件发送失败: " + err.Error())
		return
	}
	if success {
		s.appendLog("已发送文件夹压缩备份完成通知邮件至 " + cfg.NotifyEmail)
	} else {
		s.appendLog("已发送文件夹压缩备份异常通知邮件至 " + cfg.NotifyEmail)
	}
}

func (s *FolderZipBackupService) appendLog(line string) {
	s.mu.Lock()
	s.logs = append(s.logs, line)
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
	s.mu.Unlock()
	s.app.Event.Emit("folderzip-log", line)
}
