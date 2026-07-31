package zipbak

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"z-server-backup-tools/backend/util"
)

func InitState(sourceDir, statePath, stagingDir, partPrefix string) (*State, error) {
	sourceDir = filepath.Clean(sourceDir)
	stagingDir = filepath.Clean(stagingDir)
	statePath = NormalizeStatePath(statePath)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, err
	}
	files, err := scanManifest(sourceDir)
	if err != nil {
		return nil, err
	}
	store, err := OpenStore(statePath)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	if err := store.Init(sourceDir, stagingDir, partPrefix, files); err != nil {
		return nil, err
	}
	return &State{
		SourceDir:     sourceDir,
		StagingDir:    stagingDir,
		NextFileIndex: 0,
		PartSerial:    0,
		Done:          len(files) == 0,
	}, nil
}

func scanManifest(root string) ([]ManifestEntry, error) {
	var files []ManifestEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, ManifestEntry{
			RelPath:   util.ToSlashRel(rel),
			SizeBytes: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	return files, nil
}
