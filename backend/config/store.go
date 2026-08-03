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
	SkippedUpdateVersion   string                    `json:"skipped_update_version,omitempty"`
	EmailConfig            model.NotifyConfig        `json:"email_config,omitempty"`
	Servers                []model.Server            `json:"servers,omitempty"`
	BackupTasks            []model.BackupTask        `json:"backup_tasks,omitempty"`
	ActiveTaskID           string                    `json:"active_task_id,omitempty"`
	ActiveSingleFileTaskID string                    `json:"active_single_file_task_id,omitempty"`
	BackupTiming           model.BackupTimingSession `json:"backup_timing,omitempty"`
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
	var payload diskConfig
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	s.SkippedUpdateVersion = payload.SkippedUpdateVersion
	s.Backup = payload.EmailConfig.BackupConfig()
	s.Servers = payload.Servers
	s.BackupTasks = payload.BackupTasks
	s.ActiveTaskID = payload.ActiveTaskID
	s.ActiveSingleFileTaskID = payload.ActiveSingleFileTaskID
	s.BackupTiming = payload.BackupTiming
	return nil
}

func (s *Store) save() error {
	payload := diskConfig{
		SkippedUpdateVersion:   s.SkippedUpdateVersion,
		EmailConfig:            model.NotifyConfigFrom(s.Backup),
		Servers:                s.Servers,
		BackupTasks:            s.BackupTasks,
		ActiveTaskID:           s.ActiveTaskID,
		ActiveSingleFileTaskID: s.ActiveSingleFileTaskID,
		BackupTiming:           s.BackupTiming,
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
