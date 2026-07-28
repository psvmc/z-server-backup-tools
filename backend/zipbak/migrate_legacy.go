package zipbak

import (
	"os"
	"path/filepath"
	"strings"
)

func migrateLegacyStateFiles(dbPath string) error {
	if _, err := os.Stat(dbPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	appDir := appDirFromDataStateDB(dbPath)
	if appDir == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	oldDB := filepath.Join(appDir, "state.db")
	if _, err := os.Stat(oldDB); err == nil {
		return os.Rename(oldDB, dbPath)
	}
	return nil
}

func appDirFromDataStateDB(dbPath string) string {
	dir := filepath.Dir(dbPath)
	if strings.EqualFold(filepath.Base(dir), "data") {
		return filepath.Dir(dir)
	}
	return ""
}

func legacyJSONPath(dbPath string) string {
	if strings.HasSuffix(strings.ToLower(dbPath), ".json") {
		return dbPath
	}
	if app := appDirFromDataStateDB(dbPath); app != "" {
		rootJSON := filepath.Join(app, "state.json")
		if _, err := os.Stat(rootJSON); err == nil {
			return rootJSON
		}
	}
	if strings.HasSuffix(strings.ToLower(dbPath), ".db") {
		return strings.TrimSuffix(dbPath, filepath.Ext(dbPath)) + ".json"
	}
	return dbPath + ".json"
}
