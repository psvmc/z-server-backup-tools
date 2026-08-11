package service

import (
	"strings"
	"testing"

	"z-server-backup-tools/backend/model"
)

func TestResolveFolderZipTaskQueue_PreservesOrder(t *testing.T) {
	tasks := []model.BackupTask{
		{ID: "a", Kind: "folder_zip", ServerID: "s1", RemoteSource: "/a", LocalDir: "/out"},
		{ID: "b", Kind: "folder_zip", ServerID: "s1", RemoteSource: "/b", LocalDir: "/out"},
		{ID: "c", Kind: "single", ServerID: "s1", RemoteSource: "/x", LocalDir: "/out"},
		{ID: "d", Kind: "folder_zip", ServerID: "s1", RemoteSource: "/d", LocalDir: "/out"},
	}
	got, err := resolveFolderZipTaskQueue(tasks, []string{"d", "a"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"d", "a"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestResolveFolderZipTaskQueue_RejectsInvalid(t *testing.T) {
	tasks := []model.BackupTask{
		{ID: "a", Kind: "folder_zip", ServerID: "s1", RemoteSource: "/a", LocalDir: "/out"},
		{ID: "b", Kind: "folder_zip", RemoteSource: "/b", LocalDir: "/out"},
	}
	_, err := resolveFolderZipTaskQueue(tasks, []string{"b"})
	if err == nil || !strings.Contains(err.Error(), "配置不完整") {
		t.Fatalf("expected incomplete config error, got %v", err)
	}
}

func TestResolveFolderZipTaskQueue_EmptySelection(t *testing.T) {
	_, err := resolveFolderZipTaskQueue(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
