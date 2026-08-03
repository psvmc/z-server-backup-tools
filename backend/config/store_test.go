package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"z-server-backup-tools/backend/model"
)

func TestStoreSaveEmailConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	s := &Store{filePath: path}
	s.Backup = model.BackupConfig{
		NotifyEmail:  "a@b.c",
		SmtpHost:     "smtp.qq.com",
		SmtpPort:     465,
		SmtpUser:     "user",
		SmtpPassword: "pass",
	}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"email_config"`) {
		t.Fatalf("expected email_config in config file:\n%s", data)
	}
	var payload diskConfig
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.EmailConfig.NotifyEmail != "a@b.c" || payload.EmailConfig.SmtpHost != "smtp.qq.com" || payload.EmailConfig.SmtpPort != 465 {
		t.Fatalf("unexpected email_config: %+v", payload.EmailConfig)
	}

	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if s2.Backup.NotifyEmail != "a@b.c" || s2.Backup.SmtpPassword != "pass" {
		t.Fatalf("unexpected loaded email config: %+v", s2.Backup)
	}
}

func TestStoreServersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.Servers = []model.Server{{
		ID: "s1", Name: "生产", Host: "1.1.1.1", Port: 22, User: "u",
		SupportMultiFile: true, RemoteAppDir: `D:\z`, MaxPartGB: 2,
	}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Servers) != 1 || s2.Servers[0].ID != "s1" {
		t.Fatalf("got %+v", s2.Servers)
	}
}

func TestStoreDeleteServerBlockedWhenReferenced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.Servers = []model.Server{{ID: "s1", Name: "a", Host: "h", User: "u", Port: 22}}
	s.BackupTasks = []model.BackupTask{{ID: "t1", ServerID: "s1", RemoteSource: `D:\a`, LocalDir: `C:\b`}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteServer("s1"); err == nil {
		t.Fatal("expected delete blocked by task reference")
	}
	s.BackupTasks = nil
	_ = s.save()
	if err := s.DeleteServer("s1"); err != nil {
		t.Fatal("expected delete allowed when server is not referenced by tasks")
	}
	if len(s.GetServers()) != 0 {
		t.Fatal("expected empty")
	}
}

func TestStoreActiveSingleFileTaskIDRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.ActiveSingleFileTaskID = "sf1"
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if s2.GetActiveSingleFileTaskID() != "sf1" {
		t.Fatalf("got %q", s2.GetActiveSingleFileTaskID())
	}
}

func TestStoreDeleteServerBlockedBySingleKindTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.Servers = []model.Server{{ID: "s1", Name: "a", Host: "h", User: "u", Port: 22}}
	s.BackupTasks = []model.BackupTask{{
		ID: "t1", Kind: "single", ServerID: "s1",
		RemoteSource: `/var/a.bak`, LocalDir: `C:\b`,
	}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteServer("s1"); err == nil {
		t.Fatal("expected delete blocked by single task")
	}
}

func TestStoreBackupTimingRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.BackupTiming = model.BackupTimingSession{
		StartedAtMs:        1000,
		PackedFilesAtStart: 2,
		EstimatedTotalMs:   5000,
	}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"started_at_ms"`) {
		t.Fatalf("expected snake_case timing fields:\n%s", data)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if !s2.BackupTiming.Active() || s2.BackupTiming.StartedAtMs != 1000 || s2.BackupTiming.PackedFilesAtStart != 2 {
		t.Fatalf("unexpected timing: %+v", s2.BackupTiming)
	}
}
