package zipbak

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"z-server-backup-tools/backend/util"
)

func Pack(statePath string, maxPartBytes int64) (zipPath string, err error) {
	if maxPartBytes <= 0 {
		return "", fmt.Errorf("max part size must be positive")
	}
	statePath = NormalizeStatePath(statePath)
	store, err := OpenStore(statePath)
	if err != nil {
		return "", err
	}
	defer store.Close()

	m, err := store.loadMeta()
	if err != nil {
		return "", err
	}
	if err := store.validateMeta(m); err != nil {
		return "", err
	}
	if m.Done {
		return "", fmt.Errorf("备份已完成")
	}
	if m.PendingZip != "" {
		if _, err := os.Stat(m.PendingZip); err == nil {
			return m.PendingZip, nil
		}
		m.PendingZip = ""
	}
	if m.NextFileIndex >= m.FileCount {
		m.Done = true
		_ = store.updateMeta(m)
		return "", fmt.Errorf("备份已完成")
	}

	zipPath, meta, err := buildPartZip(store, m, maxPartBytes)
	if err != nil {
		return "", err
	}
	meta.PendingZip = zipPath
	if err := store.updateMeta(meta); err != nil {
		return zipPath, err
	}
	return zipPath, nil
}

func buildPartZip(store *Store, m metaRow, maxPartBytes int64) (zipPath string, updated metaRow, err error) {
	batch, nextIdx, oversized, err := selectBatch(store, m, maxPartBytes)
	if err != nil {
		return "", m, err
	}
	if len(batch) == 0 {
		return "", m, fmt.Errorf("没有可打包的文件")
	}
	for _, o := range oversized {
		fmt.Fprintf(os.Stderr, "警告: 文件 %s (%s) 超过分卷上限 (%s)，仍将单独打包\n",
			o.RelPath, util.FormatBytes(o.Size), util.FormatBytes(maxPartBytes))
	}

	m.PartSerial++
	name := PartZipName(m.PartNamePrefix, m.PartSerial)
	zipPath = filepath.Join(m.StagingDir, name)
	if err := os.MkdirAll(m.StagingDir, 0o755); err != nil {
		return "", m, err
	}
	if err := writeZip(zipPath, m.SourceDir, batch); err != nil {
		return "", m, err
	}

	m.NextFileIndex = nextIdx
	if nextIdx >= m.FileCount {
		m.Done = true
	}
	return zipPath, m, nil
}

func selectBatch(store *Store, m metaRow, maxPartBytes int64) (rels []string, nextIdx int, oversized []OversizedFile, err error) {
	rows, err := store.db.Query(`SELECT seq, rel_path, size_bytes FROM files WHERE seq >= ? ORDER BY seq`, m.NextFileIndex)
	if err != nil {
		return nil, m.NextFileIndex, nil, err
	}
	defer rows.Close()

	idx := m.NextFileIndex
	var total int64
	for rows.Next() {
		var seq int
		var rel string
		var storedSize int64
		if err := rows.Scan(&seq, &rel, &storedSize); err != nil {
			return nil, idx, nil, err
		}
		if seq != idx {
			return nil, idx, nil, fmt.Errorf("文件序号不连续: 期望 %d 得到 %d", idx, seq)
		}
		size, statErr := fileSizeFromStore(storedSize, m.SourceDir, rel)
		if statErr != nil {
			return nil, idx, nil, statErr
		}
		if len(rels) == 0 && size > maxPartBytes {
			rels = append(rels, rel)
			oversized = append(oversized, OversizedFile{RelPath: rel, Size: size})
			return rels, idx + 1, oversized, nil
		}
		if len(rels) > 0 && total+size > maxPartBytes {
			return rels, idx, oversized, nil
		}
		rels = append(rels, rel)
		total += size
		idx++
		if total >= maxPartBytes {
			return rels, idx, oversized, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, idx, nil, err
	}
	return rels, idx, oversized, nil
}

func writeZip(zipPath, sourceRoot string, rels []string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, rel := range rels {
		if err := addFileToZip(zw, sourceRoot, rel); err != nil {
			zw.Close()
			return err
		}
	}
	return zw.Close()
}

func addFileToZip(zw *zip.Writer, sourceRoot, rel string) error {
	abs := filepath.Join(sourceRoot, filepath.FromSlash(rel))
	src, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer src.Close()
	w, err := zw.Create(rel)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

func Ack(statePath string) error {
	statePath = NormalizeStatePath(statePath)
	store, err := OpenStore(statePath)
	if err != nil {
		return err
	}
	defer store.Close()
	m, err := store.loadMeta()
	if err != nil {
		return err
	}
	if m.PendingZip == "" {
		return fmt.Errorf("没有待确认的 zip 包")
	}
	m.PendingZip = ""
	if m.PrefetchZip != "" {
		m.PendingZip = m.PrefetchZip
		m.PrefetchZip = ""
	}
	if m.NextFileIndex >= m.FileCount && m.PendingZip == "" {
		m.Done = true
	}
	return store.updateMeta(m)
}

func ReadStatus(statePath string, maxPartBytes int64) (Status, error) {
	statePath = NormalizeStatePath(statePath)
	store, err := OpenStore(statePath)
	if err != nil {
		return Status{}, err
	}
	defer store.Close()
	return store.readStatus(maxPartBytes)
}
