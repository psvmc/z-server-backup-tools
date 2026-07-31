package remote

import (
	"strings"
	"testing"
)

func TestBuildRemoteCommandLinux(t *testing.T) {
	cmd := BuildRemoteCommand("linux", "/opt/zipbak/zipbak-srv", "status", "--state", "/opt/zipbak/data/state.db")
	if !strings.Contains(cmd, "/opt/zipbak/zipbak-srv") || !strings.Contains(cmd, "status") {
		t.Fatalf("%q", cmd)
	}
	if strings.Contains(cmd, `"`) {
		t.Fatalf("linux command should not use Windows double quotes: %q", cmd)
	}
}

func TestBuildRemoteCommandWindows(t *testing.T) {
	cmd := BuildRemoteCommand("windows", `D:\Tools\zipbak\zipbak-srv.exe`, "status", "--state", `D:\Tools\zipbak\data\state.db`)
	if !strings.Contains(cmd, `D:\Tools\zipbak\zipbak-srv.exe`) || !strings.Contains(cmd, "status") {
		t.Fatalf("%q", cmd)
	}
}

func TestBuildRemoteCommandLinuxQuotesSpaces(t *testing.T) {
	cmd := BuildRemoteCommand("linux", "/opt/my app/zipbak-srv", "status", "--state", "/opt/my app/data/state.db")
	if !strings.Contains(cmd, `'/opt/my app/zipbak-srv'`) {
		t.Fatalf("expected shell single quotes: %q", cmd)
	}
	if strings.Contains(cmd, `"`) {
		t.Fatalf("linux command should not use Windows double quotes: %q", cmd)
	}
}
