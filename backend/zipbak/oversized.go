package zipbak

import (
	"encoding/json"
	"fmt"

	"z-server-backup-tools/backend/util"
)

// OversizedFile is a manifest entry larger than the configured max part size.
type OversizedFile struct {
	RelPath string `json:"relPath"`
	Size    int64  `json:"size"`
}

// ListOversizedFiles returns manifest entries larger than maxPartBytes (uses init-time sizes).
func ListOversizedFiles(statePath string, maxPartBytes int64) ([]OversizedFile, error) {
	if maxPartBytes <= 0 {
		return nil, fmt.Errorf("max part size must be positive")
	}
	statePath = NormalizeStatePath(statePath)
	store, err := OpenStore(statePath)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	return store.listOversizedFiles(maxPartBytes)
}

func (s *Store) listOversizedFiles(maxPartBytes int64) ([]OversizedFile, error) {
	m, err := s.loadMeta()
	if err != nil {
		return nil, err
	}
	if err := s.validateMeta(m); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT rel_path, size_bytes FROM files WHERE size_bytes > ? ORDER BY seq`, maxPartBytes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OversizedFile
	for rows.Next() {
		var rel string
		var size int64
		if err := rows.Scan(&rel, &size); err != nil {
			return nil, err
		}
		out = append(out, OversizedFile{RelPath: rel, Size: size})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	legacy, err := s.db.Query(`SELECT rel_path FROM files WHERE size_bytes = 0 ORDER BY seq`)
	if err != nil {
		return nil, err
	}
	defer legacy.Close()
	for legacy.Next() {
		var rel string
		if err := legacy.Scan(&rel); err != nil {
			return nil, err
		}
		size, err := fileSizeFromStore(0, m.SourceDir, rel)
		if err != nil {
			return nil, err
		}
		if size > maxPartBytes {
			out = append(out, OversizedFile{RelPath: rel, Size: size})
		}
	}
	if err := legacy.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ParseOversizedJSON decodes output from zipbak-srv oversized.
func ParseOversizedJSON(raw string) ([]OversizedFile, error) {
	var items []OversizedFile
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// OversizedWarningLines builds task log lines for files exceeding the part limit.
func OversizedWarningLines(maxPartBytes int64, files []OversizedFile) []string {
	if len(files) == 0 {
		return nil
	}
	lines := []string{
		fmt.Sprintf(
			"警告: 分卷上限为 %s，以下 %d 个文件超过上限，仍将单独打包成卷",
			util.FormatBytes(maxPartBytes),
			len(files),
		),
	}
	const maxListed = 15
	for i, f := range files {
		if i >= maxListed {
			lines = append(lines, fmt.Sprintf("  … 另有 %d 个未列出", len(files)-maxListed))
			break
		}
		lines = append(lines, fmt.Sprintf("  · %s (%s)", f.RelPath, util.FormatBytes(f.Size)))
	}
	return lines
}
