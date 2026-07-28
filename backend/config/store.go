package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"z-server-backup-tools/backend/model"
)

const appConfigDir = "z-server-backup-tools"

type diskConfig struct {
	SkippedUpdateVersion string                    `json:"skippedUpdateVersion,omitempty"`
	Backup               model.BackupConfig        `json:"backup"`
	BackupTiming         model.BackupTimingSession `json:"backupTiming,omitempty"`
}

type Store struct {
	mu                   sync.Mutex
	filePath             string
	SkippedUpdateVersion string
	Backup               model.BackupConfig
	BackupTiming         model.BackupTimingSession
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
	store := &Store{filePath: filepath.Join(appDir, "config.json")}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if store.Backup.Port == 0 {
		store.Backup.Port = 22
	}
	if store.Backup.MaxPartGB == 0 {
		store.Backup.MaxPartGB = 2
	}
	return store, nil
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
	s.Backup = payload.Backup
	s.BackupTiming = payload.BackupTiming
	return nil
}

func (s *Store) save() error {
	payload := diskConfig{
		SkippedUpdateVersion: s.SkippedUpdateVersion,
		Backup:               s.Backup,
		BackupTiming:         s.BackupTiming,
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
	s.Backup = cfg
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
