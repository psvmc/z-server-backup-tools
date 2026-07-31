package zipbak

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreInitPackAckStatus(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	staging := filepath.Join(dir, "staging")
	stateDB := filepath.Join(dir, "state.db")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "b.txt", "sub/c.txt"} {
		p := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := InitState(source, stateDB, staging, "")
	if err != nil {
		t.Fatal(err)
	}
	maxPart := int64(1 << 20)
	st, err := ReadStatus(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalFiles != 3 || st.PackedFiles != 0 {
		t.Fatalf("status after init: %+v", st)
	}

	zipPath, err := Pack(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if zipPath == "" {
		t.Fatal("expected zip path")
	}
	st, err = ReadStatus(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if st.PendingZip == "" || st.PackedFiles != 3 {
		t.Fatalf("after pack: %+v", st)
	}

	if err := Ack(stateDB); err != nil {
		t.Fatal(err)
	}
	st, err = ReadStatus(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if st.PendingZip != "" || !st.Done {
		t.Fatalf("after ack: %+v", st)
	}
}

func TestPackAheadPromoteOnAck(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	staging := filepath.Join(dir, "staging")
	stateDB := filepath.Join(dir, "state.db")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("x"), 100)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		p := filepath.Join(source, name)
		if err := os.WriteFile(p, payload, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := InitState(source, stateDB, staging, ""); err != nil {
		t.Fatal(err)
	}
	maxPart := int64(150)

	zip1, err := Pack(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	zip2, err := PackAhead(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if zip2 == "" {
		t.Fatal("expected prefetch zip")
	}
	st, err := ReadStatus(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if st.PendingZip != zip1 || st.PrefetchZip != zip2 {
		t.Fatalf("status: %+v", st)
	}

	if err := Ack(stateDB); err != nil {
		t.Fatal(err)
	}
	st, err = ReadStatus(stateDB, maxPart)
	if err != nil {
		t.Fatal(err)
	}
	if st.PendingZip != zip2 || st.PrefetchZip != "" {
		t.Fatalf("after ack promote: %+v", st)
	}
}

func TestMigrateJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")
	dbPath := filepath.Join(dir, "state.db")
	st := &State{
		SourceDir:     filepath.Join(dir, "src"),
		StagingDir:    filepath.Join(dir, "staging"),
		MaxPartBytes:  1024,
		Files:         []string{"a.txt", "b.txt"},
		NextFileIndex: 1,
		PartSerial:    1,
		Done:          false,
		PendingZip:    "",
	}
	if err := os.MkdirAll(st.SourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range st.Files {
		p := filepath.Join(st.SourceDir, filepath.FromSlash(name))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(jsonPath)
	_ = data
	// write json manually
	if err := os.WriteFile(jsonPath, []byte(`{
  "source_dir": "`+filepath.ToSlash(st.SourceDir)+`",
  "staging_dir": "`+filepath.ToSlash(st.StagingDir)+`",
  "max_part_bytes": 1024,
  "files": ["a.txt","b.txt"],
  "next_file_index": 1,
  "part_serial": 1,
  "done": false
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	status, err := store.readStatus(1024)
	if err != nil {
		t.Fatal(err)
	}
	if status.TotalFiles != 2 || status.PackedFiles != 1 {
		t.Fatalf("migrated status: %+v", status)
	}
	if _, err := os.Stat(jsonPath + ".bak"); err != nil {
		t.Fatalf("expected json backup: %v", err)
	}
}
