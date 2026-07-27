package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"z-server-backup-tools/backend/model"
)

func TestStoreSaveBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	s := &Store{filePath: path}
	s.Backup = model.BackupConfig{
		Host:         "192.168.1.1",
		Port:         22,
		User:         "admin",
		Password:     "secret",
		RemoteAppDir: "D:/Tools/zipbak",
		RemoteSource: "D:/data",
		LocalDir:     "C:/backup",
		MaxPartGB:    2,
	}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload diskConfig
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Backup.Host != "192.168.1.1" || payload.Backup.User != "admin" {
		t.Fatalf("unexpected backup: %+v", payload.Backup)
	}

	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if s2.Backup.Password != "secret" {
		t.Fatalf("password not loaded: %q", s2.Backup.Password)
	}
}
