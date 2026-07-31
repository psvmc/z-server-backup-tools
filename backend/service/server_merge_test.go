package service

import (
	"testing"

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
