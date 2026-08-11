package remote

import (
	"strings"
	"testing"
)

func TestBuildLinuxCompressListCmd(t *testing.T) {
	cmd, err := buildLinuxCompressListCmd("/var/www", "/tmp/out.zip", "/tmp/list.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "cd /var/www") || !strings.Contains(cmd, "zip -q /tmp/out.zip -@") || !strings.Contains(cmd, "/tmp/list.txt") {
		t.Fatalf("unexpected cmd: %s", cmd)
	}
}

func TestBuildWindowsCompressListCmd(t *testing.T) {
	cmd, err := buildWindowsCompressListCmd(`D:\data`, `C:\Temp\out.zip`, `C:\Temp\list.txt`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, `tar -a -c -f C:\Temp\out.zip`) || !strings.Contains(cmd, `-C D:\data`) || !strings.Contains(cmd, `-T C:\Temp\list.txt`) {
		t.Fatalf("unexpected cmd: %s", cmd)
	}
}

func TestBuildLinuxCompressCmd(t *testing.T) {
	cmd, err := buildLinuxCompressCmd("/var/www/myapp", "/tmp/out.zip")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "cd /var/www") || !strings.Contains(cmd, "zip -r -q /tmp/out.zip myapp") {
		t.Fatalf("unexpected cmd: %s", cmd)
	}
}

func TestBuildWindowsCompressCmd(t *testing.T) {
	cmd, err := buildWindowsCompressCmd(`D:\data\myapp`, `C:\Temp\out.zip`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, `tar -a -c -f C:\Temp\out.zip`) || !strings.Contains(cmd, `-C D:\data myapp`) {
		t.Fatalf("unexpected cmd: %s", cmd)
	}
}

func TestBuildRemoteTempZipPath(t *testing.T) {
	if got := BuildRemoteTempZipPath("linux", "myapp", "abc"); got != "/tmp/z-srv-backup-abc-myapp.zip" {
		t.Fatalf("linux: %q", got)
	}
	if got := BuildRemoteTempZipPath("windows", "myapp", "abc"); got != `%TEMP%\z-srv-backup-abc-myapp.zip` {
		t.Fatalf("windows: %q", got)
	}
}
