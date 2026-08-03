package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"z-server-backup-tools/backend/model"
)

const appConfigDir = "z-server-backup-tools"

type diskConfig struct {
	SkippedUpdateVersion   string                    `json:"skippedUpdateVersion,omitempty"`
	Backup                 model.NotifyConfig        `json:"backup,omitempty"`
	Servers                []model.Server            `json:"servers,omitempty"`
	BackupTasks            []model.BackupTask        `json:"backupTasks,omitempty"`
	ActiveTaskID           string                    `json:"activeTaskId,omitempty"`
	ActiveSingleFileTaskID string                    `json:"activeSingleFileTaskId,omitempty"`
	BackupTiming           model.BackupTimingSession `json:"backupTiming,omitempty"`
	SingleFile             model.SingleFileConfig    `json:"singleFileBackup,omitempty"`
}

type Store struct {
	mu                     sync.Mutex
	filePath               string
	SkippedUpdateVersion   string
	Backup                 model.BackupConfig
	Servers                []model.Server
	BackupTasks            []model.BackupTask
	ActiveTaskID           string
	ActiveSingleFileTaskID string
	BackupTiming           model.BackupTimingSession
	SingleFile             model.SingleFileConfig
}

var (
	defaultStore    *Store
	defaultStoreErr error
	storeInit       sync.Once
)

// Default returns the process-wide config store (shared by all services).
func Default() (*Store, error) {
	storeInit.Do(func() {
		defaultStore, defaultStoreErr = NewStore()
	})
	return defaultStore, defaultStoreErr
}

func NewStore() (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(configDir, appConfigDir)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, err
	}
	store := NewStoreAt(filepath.Join(appDir, "config.json"))
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}

// NewStoreAt returns a store backed by the given config file path.
func NewStoreAt(filePath string) *Store {
	return &Store{filePath: filePath}
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	var envelope struct {
		SkippedUpdateVersion   string                    `json:"skippedUpdateVersion,omitempty"`
		Backup                 json.RawMessage           `json:"backup"`
		Servers                []model.Server            `json:"servers,omitempty"`
		BackupTasks            []model.BackupTask        `json:"backupTasks,omitempty"`
		ActiveTaskID           string                    `json:"activeTaskId,omitempty"`
		ActiveSingleFileTaskID string                    `json:"activeSingleFileTaskId,omitempty"`
		BackupTiming           model.BackupTimingSession `json:"backupTiming,omitempty"`
		SingleFile             model.SingleFileConfig    `json:"singleFileBackup,omitempty"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	var legacyBackup model.BackupConfig
	if len(envelope.Backup) > 0 {
		if err := json.Unmarshal(envelope.Backup, &legacyBackup); err != nil {
			return err
		}
	}
	s.SkippedUpdateVersion = envelope.SkippedUpdateVersion
	s.Backup = legacyBackup
	s.Servers = envelope.Servers
	s.BackupTasks = envelope.BackupTasks
	s.ActiveTaskID = envelope.ActiveTaskID
	s.ActiveSingleFileTaskID = envelope.ActiveSingleFileTaskID
	s.BackupTiming = envelope.BackupTiming
	s.SingleFile = envelope.SingleFile
	migrated := s.migrateLegacyTask()
	cleaned := s.Backup.NotifyOnly()
	needsSave := migrated || s.Backup != cleaned
	s.Backup = cleaned
	if needsSave {
		return s.save()
	}
	return nil
}

func (s *Store) migrateLegacyTask() bool {
	if len(s.BackupTasks) > 0 {
		return false
	}
	src := strings.TrimSpace(s.Backup.RemoteSource)
	local := strings.TrimSpace(s.Backup.LocalDir)
	prefix := strings.TrimSpace(s.Backup.PartNamePrefix)
	if src == "" && local == "" && prefix == "" {
		return false
	}
	task := model.BackupTask{
		ID:             "default",
		Name:           "默认任务",
		RemoteSource:   src,
		LocalDir:       local,
		PartNamePrefix: prefix,
	}
	s.BackupTasks = []model.BackupTask{task}
	if strings.TrimSpace(s.ActiveTaskID) == "" {
		s.ActiveTaskID = task.ID
	}
	s.Backup.RemoteSource = ""
	s.Backup.LocalDir = ""
	s.Backup.PartNamePrefix = ""
	s.Backup.TaskID = ""
	return true
}

func (s *Store) save() error {
	payload := diskConfig{
		SkippedUpdateVersion:   s.SkippedUpdateVersion,
		Backup:                 model.NotifyConfigFrom(s.Backup),
		Servers:                s.Servers,
		BackupTasks:            s.BackupTasks,
		ActiveTaskID:           s.ActiveTaskID,
		ActiveSingleFileTaskID: s.ActiveSingleFileTaskID,
		BackupTiming:           s.BackupTiming,
		SingleFile:             s.SingleFile,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func (s *Store) GetSkippedUpdateVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.SkippedUpdateVersion)
}

func (s *Store) SetSkippedUpdateVersion(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.SkippedUpdateVersion = strings.TrimSpace(version)
	return s.save()
}

func (s *Store) GetBackupConfig() model.BackupConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Backup
}

func (s *Store) SetBackupConfig(cfg model.BackupConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.Backup = cfg.NotifyOnly()
	return s.save()
}

func (s *Store) ConfigPath() string {
	return s.filePath
}

func (s *Store) GetBackupTiming() model.BackupTimingSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.BackupTiming
}

func (s *Store) SetBackupTiming(session model.BackupTimingSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.BackupTiming = session
	return s.save()
}

func (s *Store) ClearBackupTiming() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.BackupTiming = model.BackupTimingSession{}
	return s.save()
}

func (s *Store) GetBackupTasks() []model.BackupTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.BackupTask, len(s.BackupTasks))
	copy(out, s.BackupTasks)
	return out
}

func (s *Store) SetBackupTasks(tasks []model.BackupTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.BackupTasks = tasks
	return s.save()
}

func (s *Store) GetActiveTaskID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.ActiveTaskID)
}

func (s *Store) SetActiveTaskID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.ActiveTaskID = strings.TrimSpace(id)
	return s.save()
}

func (s *Store) GetActiveSingleFileTaskID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.ActiveSingleFileTaskID)
}

func (s *Store) SetActiveSingleFileTaskID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.ActiveSingleFileTaskID = strings.TrimSpace(id)
	return s.save()
}

func (s *Store) GetSingleFileConfig() model.SingleFileConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.SingleFile
}

func (s *Store) SetSingleFileConfig(cfg model.SingleFileConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.SingleFile = cfg
	return s.save()
}

func (s *Store) GetServers() []model.Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Server, len(s.Servers))
	copy(out, s.Servers)
	return out
}

func (s *Store) SetServers(servers []model.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.Servers = servers
	return s.save()
}

func (s *Store) DeleteServer(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, task := range s.BackupTasks {
		if task.ServerID == id {
			return fmt.Errorf("服务器仍被引用，无法删除")
		}
	}
	out := make([]model.Server, 0, len(s.Servers))
	for _, srv := range s.Servers {
		if srv.ID != id {
			out = append(out, srv)
		}
	}
	s.Servers = out
	return s.save()
}
