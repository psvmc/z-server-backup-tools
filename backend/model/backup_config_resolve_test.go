package model

import "testing"

func TestResolvedLinuxSrvName(t *testing.T) {
	cfg := BackupConfig{RemoteAppDir: "/opt/zipbak", OSType: OSTypeLinux, TaskID: "abc"}
	out := cfg.Resolved()
	if out.RemoteSrv != "/opt/zipbak/zipbak-srv" {
		t.Fatalf("RemoteSrv=%q", out.RemoteSrv)
	}
	if out.RemoteState != "/opt/zipbak/data/state-abc.db" {
		t.Fatalf("RemoteState=%q", out.RemoteState)
	}
	if out.RemoteStaging != "/opt/zipbak/staging-abc" {
		t.Fatalf("RemoteStaging=%q", out.RemoteStaging)
	}
}

func TestResolvedWindowsSrvName(t *testing.T) {
	cfg := BackupConfig{RemoteAppDir: `D:\Tools\zipbak`, OSType: OSTypeWindows, TaskID: "abc"}
	out := cfg.Resolved()
	if out.RemoteSrv != `D:\Tools\zipbak\zipbak-srv.exe` {
		t.Fatalf("RemoteSrv=%q", out.RemoteSrv)
	}
}
