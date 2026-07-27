package zipbak

import (
	"fmt"
	"os"
	"path/filepath"
)

// ManifestEntry is one file in the backup manifest (relative path + size at init time).
type ManifestEntry struct {
	RelPath   string
	SizeBytes int64
}

func manifestFromRelPaths(sourceDir string, rels []string) ([]ManifestEntry, error) {
	sourceDir = filepath.Clean(sourceDir)
	out := make([]ManifestEntry, 0, len(rels))
	for _, rel := range rels {
		abs := filepath.Join(sourceDir, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", rel, err)
		}
		out = append(out, ManifestEntry{RelPath: rel, SizeBytes: info.Size()})
	}
	return out, nil
}

func fileSizeFromStore(stored int64, sourceDir, rel string) (int64, error) {
	if stored > 0 {
		return stored, nil
	}
	abs := filepath.Join(sourceDir, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", rel, err)
	}
	return info.Size(), nil
}
