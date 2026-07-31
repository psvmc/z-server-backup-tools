package zipbak

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func migrateJSONIfNeeded(dbPath string) error {
	if _, err := os.Stat(dbPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	jsonPath := legacyJSONPath(dbPath)
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	st, err := loadStateJSON(jsonPath)
	if err != nil {
		return fmt.Errorf("迁移 state.json 失败: %w", err)
	}
	store, err := openStore(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	entries, err := manifestFromRelPaths(st.SourceDir, st.Files)
	if err != nil {
		return fmt.Errorf("迁移 state.json 失败: %w", err)
	}
	if err := store.Init(st.SourceDir, st.StagingDir, "", entries); err != nil {
		return err
	}
	m, err := store.loadMeta()
	if err != nil {
		return err
	}
	m.NextFileIndex = st.NextFileIndex
	m.PendingZip = st.PendingZip
	m.PartSerial = st.PartSerial
	m.Done = st.Done
	if err := store.updateMeta(m); err != nil {
		return err
	}
	backup := jsonPath + ".bak"
	_ = os.Remove(backup)
	return os.Rename(jsonPath, backup)
}

func loadStateJSON(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// NormalizeStatePath maps legacy *.json paths to *.db.
func NormalizeStatePath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasSuffix(strings.ToLower(p), ".json") {
		ext := filepath.Ext(p)
		return strings.TrimSuffix(p, ext) + ".db"
	}
	if !strings.HasSuffix(strings.ToLower(p), ".db") && p != "" {
		return p + ".db"
	}
	return p
}
