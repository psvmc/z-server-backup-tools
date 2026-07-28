package model

// BackupTimingSession is persisted locally for elapsed time across app restarts.
type BackupTimingSession struct {
	StartedAtMs        int64 `json:"startedAtMs"`
	PackedFilesAtStart int   `json:"packedFilesAtStart"`
	EstimatedTotalMs   int64 `json:"estimatedTotalMs,omitempty"`
}

func (s BackupTimingSession) Active() bool {
	return s.StartedAtMs > 0
}
