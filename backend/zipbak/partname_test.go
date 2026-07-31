package zipbak

import "testing"

func TestPartZipName(t *testing.T) {
	if got := PartZipName("", 1); got != "part-000001.zip" {
		t.Fatalf("empty prefix: %q", got)
	}
	if got := PartZipName("srv1-", 42); got != "srv1-part-000042.zip" {
		t.Fatalf("with prefix: %q", got)
	}
}

func TestIsBackupPartName(t *testing.T) {
	if !IsBackupPartName("part-000001.zip", "") {
		t.Fatal("default part name")
	}
	if !IsBackupPartName("srv1-part-000001.zip", "srv1-") {
		t.Fatal("prefixed part name")
	}
	if IsBackupPartName("part-000001.zip", "srv1-") {
		t.Fatal("wrong prefix should not match")
	}
}

func TestSanitizePartPrefix(t *testing.T) {
	if got := SanitizePartPrefix("  abc/def* "); got != "abcdef" {
		t.Fatalf("sanitize: %q", got)
	}
}
