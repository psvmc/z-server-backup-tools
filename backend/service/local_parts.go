package service

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/zipbak"
)

const localPartDownloadingSuffix = ".downloading"

func listLocalPartFiles(localDir, partPrefix string) (model.LocalPartListing, error) {
	out := model.LocalPartListing{LocalDir: localDir}
	localDir = strings.TrimSpace(localDir)
	if localDir == "" {
		return out, nil
	}
	entries, err := os.ReadDir(localDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	var files []model.LocalPartFile
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		lower := strings.ToLower(name)
		state := "downloaded"
		displayName := name
		if strings.HasSuffix(lower, localPartDownloadingSuffix) {
			state = "downloading"
			displayName = strings.TrimSuffix(name, localPartDownloadingSuffix)
		}
		if !zipbak.IsBackupPartName(displayName, partPrefix) {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		files = append(files, model.LocalPartFile{
			Name:      displayName,
			Path:      filepath.Join(localDir, name),
			SizeBytes: info.Size(),
			State:     state,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	out.Files = files
	return out, nil
}

func (s *BackupService) ListLocalParts(cfg model.BackupConfig) (model.LocalPartListing, error) {
	dir := strings.TrimSpace(cfg.LocalDir)
	prefix := zipbak.SanitizePartPrefix(cfg.PartNamePrefix)
	if dir == "" {
		stored := s.storedConfig()
		dir = strings.TrimSpace(stored.LocalDir)
		if prefix == "" {
			prefix = zipbak.SanitizePartPrefix(stored.PartNamePrefix)
		}
	}
	return listLocalPartFiles(dir, prefix)
}
