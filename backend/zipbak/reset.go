package zipbak

import (
	"os"
)

// ResetProgress clears backup progress but keeps the file manifest (re-init not required).
func ResetProgress(statePath string) error {
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
	if err := store.validateMeta(m); err != nil {
		return err
	}
	if m.PendingZip != "" {
		_ = os.Remove(m.PendingZip)
	}
	m.NextFileIndex = 0
	m.PendingZip = ""
	m.PartSerial = 0
	m.Done = m.FileCount == 0
	return store.updateMeta(m)
}
