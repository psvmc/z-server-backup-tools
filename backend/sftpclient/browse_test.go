package sftpclient

import (
	"testing"

	"z-server-backup-tools/backend/model"
)

func TestToFromSFTPPathWindows(t *testing.T) {
	if got := toSFTPPath(`D:\Tools`, model.OSTypeWindows); got != "/D:/Tools" {
		t.Fatalf("windows toSFTP: %q", got)
	}
	if got := fromSFTPPath("/D:/Tools", model.OSTypeWindows); got != `D:\Tools` {
		t.Fatalf("windows fromSFTP: %q", got)
	}
}

func TestToFromSFTPPathLinux(t *testing.T) {
	if got := toSFTPPath("/var/log", model.OSTypeLinux); got != "/var/log" {
		t.Fatalf("linux toSFTP: %q", got)
	}
	if got := toSFTPPath("var/log", model.OSTypeLinux); got != "/var/log" {
		t.Fatalf("linux toSFTP relative: %q", got)
	}
	if got := fromSFTPPath("/var/log", model.OSTypeLinux); got != "/var/log" {
		t.Fatalf("linux fromSFTP: %q", got)
	}
	if got := fromSFTPPath("/", model.OSTypeLinux); got != "/" {
		t.Fatalf("linux root: %q", got)
	}
}

func TestJoinRemotePath(t *testing.T) {
	if got := JoinRemotePath("/var", "log", model.OSTypeLinux); got != "/var/log" {
		t.Fatalf("linux join: %q", got)
	}
	if got := JoinRemotePath("/", "etc", model.OSTypeLinux); got != "/etc" {
		t.Fatalf("linux join root: %q", got)
	}
	if got := JoinRemotePath(`D:\data`, "app.bak", model.OSTypeWindows); got != `D:\data\app.bak` {
		t.Fatalf("windows join: %q", got)
	}
}
