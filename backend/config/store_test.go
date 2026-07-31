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

func TestStoreSingleFileBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.SingleFile = model.SingleFileConfig{
		RemoteFile: `D:\data\app.bak`,
		LocalDir:   `C:\backup`,
	}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if s2.SingleFile.RemoteFile != `D:\data\app.bak` || s2.SingleFile.LocalDir != `C:\backup` {
		t.Fatalf("got %+v", s2.SingleFile)
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
	s.SingleFile = model.SingleFileConfig{ServerID: "s1", RemoteFile: `D:\f`, LocalDir: `C:\b`}
	_ = s.save()
	if err := s.DeleteServer("s1"); err != nil {
		t.Fatal("expected delete allowed when only legacy singleFileBackup references server")
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
