package sftpclient

import "testing"

func TestRemoteFolderBaseName(t *testing.T) {
	name, err := remoteFolderBaseName(`D:\data\myapp`)
	if err != nil || name != "myapp" {
		t.Fatalf("got %q %v", name, err)
	}
	name, err = remoteFolderBaseName(`/var/www/app/`)
	if err != nil || name != "app" {
		t.Fatalf("got %q %v", name, err)
	}
}
