package service

import (
	"path/filepath"
	"strings"
	"testing"

	"z-server-backup-tools/backend/config"
	"z-server-backup-tools/backend/model"
)

func testStore(t *testing.T) *config.Store {
	t.Helper()
	return config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
}

func TestValidateBackupTask_SingleVsMulti(t *testing.T) {
	servers := []model.Server{
		{ID: "s1", SupportMultiFile: false},
		{ID: "s2", SupportMultiFile: true},
	}
	single := model.BackupTask{
		ID: "a", Kind: "single", ServerID: "s1",
		RemoteSource: "/f.bak", LocalDir: `C:\o`,
	}
	if err := validateBackupTask(servers, single); err != nil {
		t.Fatal(err)
	}
	multiBad := model.BackupTask{
		ID: "b", ServerID: "s1",
		RemoteSource: `D:\d`, LocalDir: `C:\o`,
	}
	if err := validateBackupTask(servers, multiBad); err == nil {
		t.Fatal("multi on non-multi server should fail")
	}
	multiOK := model.BackupTask{
		ID: "c", ServerID: "s2",
		RemoteSource: `D:\d`, LocalDir: `C:\o`,
	}
	if err := validateBackupTask(servers, multiOK); err != nil {
		t.Fatal(err)
	}
}

func TestSaveTasks_SingleAllowedOnNonMultiServer(t *testing.T) {
	store := testStore(t)
	store.Servers = []model.Server{{ID: "s1", Name: "a", Host: "h", User: "u", Port: 22, SupportMultiFile: false}}
	svc := &BackupService{store: store}
	tasks := []model.BackupTask{{
		ID: "t1", Kind: "single", ServerID: "s1",
		RemoteSource: `/var/f.bak`, LocalDir: `C:\out`,
	}}
	if err := svc.SaveTasks(tasks); err != nil {
		t.Fatal(err)
	}
	saved := store.GetBackupTasks()
	if len(saved) != 1 || saved[0].Kind != "single" {
		t.Fatalf("got %+v", saved)
	}
	if saved[0].PartNamePrefix != "" {
		t.Fatalf("single task should clear prefix, got %q", saved[0].PartNamePrefix)
	}
}

func TestSaveTasks_MultiRejectsNonMultiServer(t *testing.T) {
	store := testStore(t)
	store.Servers = []model.Server{{ID: "s1", Name: "a", Host: "h", User: "u", Port: 22, SupportMultiFile: false}}
	svc := &BackupService{store: store}
	tasks := []model.BackupTask{{
		ID: "t1", ServerID: "s1",
		RemoteSource: `D:\src`, LocalDir: `C:\out`,
	}}
	err := svc.SaveTasks(tasks)
	if err == nil || !strings.Contains(err.Error(), "多文件") {
		t.Fatalf("want multi rejection, got %v", err)
	}
}

func TestSetActiveTaskID_RejectsSingle(t *testing.T) {
	store := &config.Store{
		BackupTasks: []model.BackupTask{{
			ID: "t1", Kind: "single", ServerID: "s1",
			RemoteSource: `/f.bak`, LocalDir: `C:\o`,
		}},
	}
	svc := &BackupService{store: store}
	err := svc.SetActiveTaskID("t1")
	if err == nil {
		t.Fatal("expected rejection for single task")
	}
}

func TestSetActiveTaskID_AcceptsMulti(t *testing.T) {
	store := testStore(t)
	store.BackupTasks = []model.BackupTask{{
		ID: "t1", ServerID: "s1",
		RemoteSource: `D:\src`, LocalDir: `C:\o`,
	}}
	svc := &BackupService{store: store}
	if err := svc.SetActiveTaskID("t1"); err != nil {
		t.Fatal(err)
	}
	if store.GetActiveTaskID() != "t1" {
		t.Fatalf("got %q", store.GetActiveTaskID())
	}
}

func TestSetActiveSingleFileTaskID_RejectsMulti(t *testing.T) {
	store := &config.Store{
		BackupTasks: []model.BackupTask{{
			ID: "t1", ServerID: "s1",
			RemoteSource: `D:\src`, LocalDir: `C:\o`,
		}},
	}
	svc := &BackupService{store: store}
	err := svc.SetActiveSingleFileTaskID("t1")
	if err == nil {
		t.Fatal("expected rejection for multi task")
	}
}

func TestSetActiveSingleFileTaskID_AcceptsSingle(t *testing.T) {
	store := testStore(t)
	store.BackupTasks = []model.BackupTask{{
		ID: "t1", Kind: "single", ServerID: "s1",
		RemoteSource: `/f.bak`, LocalDir: `C:\o`,
	}}
	svc := &BackupService{store: store}
	if err := svc.SetActiveSingleFileTaskID("t1"); err != nil {
		t.Fatal(err)
	}
	if store.GetActiveSingleFileTaskID() != "t1" {
		t.Fatalf("got %q", store.GetActiveSingleFileTaskID())
	}
}

func TestSaveTasks_ReconcileStaleActiveMulti(t *testing.T) {
	store := testStore(t)
	store.Servers = []model.Server{{
		ID: "s1", Name: "a", Host: "h", User: "u", Port: 22, SupportMultiFile: true, RemoteAppDir: `D:\app`, MaxPartGB: 2,
	}}
	store.ActiveTaskID = "gone"
	svc := &BackupService{store: store}
	tasks := []model.BackupTask{{
		ID: "t2", ServerID: "s1",
		RemoteSource: `D:\src`, LocalDir: `C:\out`,
	}}
	if err := svc.SaveTasks(tasks); err != nil {
		t.Fatal(err)
	}
	if store.GetActiveTaskID() != "t2" {
		t.Fatalf("expected reconcile to t2, got %q", store.GetActiveTaskID())
	}
}

func TestSaveTasks_ReconcileWrongKindActive(t *testing.T) {
	store := testStore(t)
	store.Servers = []model.Server{{
		ID: "s1", Name: "a", Host: "h", User: "u", Port: 22, SupportMultiFile: true, RemoteAppDir: `D:\app`, MaxPartGB: 2,
	}}
	store.ActiveTaskID = "sf1"
	store.BackupTasks = []model.BackupTask{{
		ID: "sf1", Kind: "single", ServerID: "s1",
		RemoteSource: `/f.bak`, LocalDir: `C:\o`,
	}}
	svc := &BackupService{store: store}
	tasks := []model.BackupTask{
		{ID: "sf1", Kind: "single", ServerID: "s1", RemoteSource: `/f.bak`, LocalDir: `C:\o`},
		{ID: "m1", ServerID: "s1", RemoteSource: `D:\src`, LocalDir: `C:\out`},
	}
	if err := svc.SaveTasks(tasks); err != nil {
		t.Fatal(err)
	}
	if store.GetActiveTaskID() != "m1" {
		t.Fatalf("expected reconcile to m1, got %q", store.GetActiveTaskID())
	}
}
