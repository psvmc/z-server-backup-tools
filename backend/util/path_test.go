package util

import "testing"

func TestJoinRemoteForOS(t *testing.T) {
	if got := JoinRemoteForOS("linux", "/opt/zipbak", "zipbak-srv"); got != "/opt/zipbak/zipbak-srv" {
		t.Fatalf("linux join srv: %q", got)
	}
	if got := JoinRemoteForOS("linux", "/opt/zipbak", "data", "state-t1.db"); got != "/opt/zipbak/data/state-t1.db" {
		t.Fatalf("linux join state: %q", got)
	}
	if got := JoinRemoteForOS("windows", `D:\Tools\zipbak`, "zipbak-srv.exe"); got != `D:\Tools\zipbak\zipbak-srv.exe` {
		t.Fatalf("windows join: %q", got)
	}
}
