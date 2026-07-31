package model

import "testing"

func TestBackupTaskNormalizedKind(t *testing.T) {
	if (BackupTask{}).NormalizedKind() != "multi" {
		t.Fatal("empty kind should be multi")
	}
	if (BackupTask{Kind: "MULTI"}).NormalizedKind() != "multi" {
		t.Fatal("case insensitive multi")
	}
	if (BackupTask{Kind: "single"}).NormalizedKind() != "single" {
		t.Fatal("single")
	}
	if (BackupTask{Kind: "other"}).NormalizedKind() != "multi" {
		t.Fatal("unknown -> multi")
	}
}
