package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListLocalPartFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "part-000001.zip"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "part-000002.zip.downloading"), []byte("ab"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	listing, err := listLocalPartFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Files) != 2 {
		t.Fatalf("files: %+v", listing.Files)
	}
	if listing.Files[0].State != "downloaded" || listing.Files[0].SizeBytes != 5 {
		t.Fatalf("first: %+v", listing.Files[0])
	}
	if listing.Files[1].State != "downloading" || listing.Files[1].Name != "part-000002.zip" {
		t.Fatalf("second: %+v", listing.Files[1])
	}
}
