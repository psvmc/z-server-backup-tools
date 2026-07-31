package service

import (
	"strings"
	"testing"

	"z-server-backup-tools/backend/config"
	"z-server-backup-tools/backend/model"
)

func TestMergeServerTaskNotify(t *testing.T) {
	notify := model.BackupConfig{NotifyEmail: "n@e.com", SmtpHost: "smtp"}
	srv := model.Server{
		ID: "s1", Host: "h", Port: 22, User: "u", Password: "p",
		SupportMultiFile: true, RemoteAppDir: `D:\app`, MaxPartGB: 2,
	}
	task := model.BackupTask{
		ID: "t1", ServerID: "s1", RemoteSource: `D:\src`, LocalDir: `C:\out`, PartNamePrefix: "p",
	}
	out := mergeServerTaskNotify(notify, srv, task)
	if out.Host != "h" || out.RemoteSource != `D:\src` || out.NotifyEmail != "n@e.com" || out.RemoteSrv == "" {
		t.Fatalf("got %+v", out)
	}
}

func TestPrepareMultiFileJobUsesStoreServer(t *testing.T) {
	store := &config.Store{
		Backup: model.BackupConfig{NotifyEmail: "n@e.com", SmtpHost: "smtp"},
		Servers: []model.Server{{
			ID: "s1", Name: "prod", Host: "store-host", Port: 22, User: "u", Password: "p",
			SupportMultiFile: true, RemoteAppDir: `D:\app`, MaxPartGB: 2,
		}},
		BackupTasks: []model.BackupTask{{
			ID: "t1", Name: "task", ServerID: "s1",
			RemoteSource: `D:\src`, LocalDir: `C:\out`, PartNamePrefix: "p",
		}},
	}
	svc := &BackupService{store: store}
	frontend := model.BackupConfig{
		TaskID: "t1",
		Host:   "evil-host", User: "evil", Password: "x",
		RemoteAppDir: `C:\fake`, RemoteSource: `D:\wrong`, LocalDir: `C:\wrong`,
	}
	prepared, err := svc.prepareMultiFileJob(frontend)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Host != "store-host" {
		t.Fatalf("host=%q want store-host (must not trust frontend SSH)", prepared.Host)
	}
	if prepared.RemoteAppDir != `D:\app` || prepared.RemoteSource != `D:\src` || prepared.LocalDir != `C:\out` {
		t.Fatalf("paths not from store merge: %+v", prepared)
	}
	if prepared.NotifyEmail != "n@e.com" || prepared.TaskID != "t1" {
		t.Fatalf("notify/task: %+v", prepared)
	}
}

func TestSaveConnectionConfigRejectsSSHWrite(t *testing.T) {
	svc := &BackupService{store: &config.Store{}}
	err := svc.SaveConnectionConfig(model.BackupConfig{Host: "h", User: "u", Password: "p"})
	if err == nil || !strings.Contains(err.Error(), "服务器管理") {
		t.Fatalf("want error directing to server management, got %v", err)
	}
}
