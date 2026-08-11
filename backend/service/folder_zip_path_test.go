package service

import (
	"path/filepath"
	"testing"
)

func TestFolderZipLocalPath(t *testing.T) {
	p, err := folderZipLocalPath(`D:\data\myapp`, `C:\out`)
	if err != nil || p != filepath.Join(`C:\out`, `myapp.zip`) {
		t.Fatalf("%q %v", p, err)
	}
}
