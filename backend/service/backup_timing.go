package service

import (
	"time"

	"z-server-backup-tools/backend/model"
)

func (s *BackupService) applyTimingToStatus(st *model.JobStatus) {
	if s.store == nil {
		return
	}
	t := s.store.GetBackupTiming()
	st.TimingStartedAtMs = t.StartedAtMs
	st.TimingPackedFilesAtStart = t.PackedFilesAtStart
	st.TimingEstimatedTotalMs = t.EstimatedTotalMs
}

func (s *BackupService) beginBackupTiming(packedFilesNow int) {
	if s.store == nil {
		return
	}
	t := s.store.GetBackupTiming()
	if t.Active() {
		return
	}
	_ = s.store.SetBackupTiming(model.BackupTimingSession{
		StartedAtMs:        time.Now().UnixMilli(),
		PackedFilesAtStart: packedFilesNow,
	})
}

func (s *BackupService) clearBackupTiming() {
	if s.store == nil {
		return
	}
	_ = s.store.ClearBackupTiming()
}

// SetBackupTimingEstimate persists ETA baseline for restore after restart.
func (s *BackupService) SetBackupTimingEstimate(estimatedTotalMs int64) error {
	if s.store == nil {
		return nil
	}
	t := s.store.GetBackupTiming()
	if !t.Active() {
		return nil
	}
	if estimatedTotalMs <= 0 {
		return nil
	}
	t.EstimatedTotalMs = estimatedTotalMs
	return s.store.SetBackupTiming(t)
}
