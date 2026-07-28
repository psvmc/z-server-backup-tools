package zipbak

import (
	"fmt"
	"os"
)

// PackAhead builds the next part zip while PendingZip is still awaiting download/ack.
// Returns ("", nil) when there is no next batch to pack.
func PackAhead(statePath string, maxPartBytes int64) (string, error) {
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
	if m.PendingZip == "" {
		return "", fmt.Errorf("pack-ahead: 尚无待下载分卷")
	}
	if m.PrefetchZip != "" {
		if _, err := os.Stat(m.PrefetchZip); err == nil {
			return m.PrefetchZip, nil
		}
		m.PrefetchZip = ""
	}
	if m.NextFileIndex >= m.FileCount {
		return "", nil
	}

	zipPath, meta, err := buildPartZip(store, m, maxPartBytes)
	if err != nil {
		return "", err
	}
	meta.PrefetchZip = zipPath
	if err := store.updateMeta(meta); err != nil {
		return zipPath, err
	}
	return zipPath, nil
}
