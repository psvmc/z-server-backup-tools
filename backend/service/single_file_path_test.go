package service

import (
	"path/filepath"
	"testing"
)

func TestSingleFileLocalPath(t *testing.T) {
	p, err := singleFileLocalPath(`D:\data\app.bak`, `C:\out`)
	if err != nil || p != filepath.Join(`C:\out`, `app.bak`) {
		t.Fatalf("%q %v", p, err)
	}
}
