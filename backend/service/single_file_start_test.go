package service

import (
	"testing"

	"z-server-backup-tools/backend/model"
)

func TestTaskToSinglePaths(t *testing.T) {
	_, err := taskToSinglePaths(model.BackupTask{Kind: "multi", ServerID: "s", RemoteSource: "a", LocalDir: "b"})
	if err == nil {
		t.Fatal("expected error")
	}
	p, err := taskToSinglePaths(model.BackupTask{
		Kind: "single", ServerID: "s", RemoteSource: `/a.bak`, LocalDir: `C:\o`,
	})
	if err != nil || p.RemoteFile != `/a.bak` {
		t.Fatalf("%+v %v", p, err)
	}
}
