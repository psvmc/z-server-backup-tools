package sftpclient

import "testing"

func TestEntryDisplayPath(t *testing.T) {
	got := entryDisplayPath("myapp", "logs/", "app.log", false, "linux")
	if got != "myapp/logs/app.log" {
		t.Fatalf("file: %q", got)
	}
	got = entryDisplayPath("myapp", "", "node_modules", true, "linux")
	if got != "myapp/node_modules/" {
		t.Fatalf("dir: %q", got)
	}
}

func TestRelPathFromParent(t *testing.T) {
	got := relPathFromParent("myapp", "logs/", "app.log", "linux")
	if got != "myapp/logs/app.log" {
		t.Fatalf("linux: %q", got)
	}
	got = relPathFromParent("myapp", "logs/", "app.log", "windows")
	if got != `myapp\logs\app.log` {
		t.Fatalf("windows: %q", got)
	}
}
