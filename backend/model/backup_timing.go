package model

// BackupTimingSession is persisted locally for elapsed time across app restarts.
type BackupTimingSession struct {
	StartedAtMs        int64 `json:"started_at_ms"`
	PackedFilesAtStart int   `json:"packed_files_at_start"`
	EstimatedTotalMs   int64 `json:"estimated_total_ms,omitempty"`
}

func (s BackupTimingSession) Active() bool {
	return s.StartedAtMs > 0
}
