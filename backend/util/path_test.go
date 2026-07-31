package util

import (
	"strings"
	"testing"
)

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

func TestQuoteShellArg(t *testing.T) {
	if got := QuoteShellArg(`hello`); got != `hello` {
		t.Fatalf("%q", got)
	}
	if got := QuoteShellArg(`a b`); got != `'a b'` {
		t.Fatalf("%q", got)
	}
	if got := QuoteShellArg(`a'b`); !strings.Contains(got, `'"'"'`) && got != `'a'"'"'b'` {
		t.Fatalf("%q", got)
	}
}
