package model

// RemoteState is persisted on the Windows source server (state.db, SQLite).
type RemoteState struct {
	SourceDir     string   `json:"source_dir"`
	StagingDir    string   `json:"staging_dir"`
	MaxPartBytes  int64    `json:"max_part_bytes"`
	Files         []string `json:"files"`
	NextFileIndex int      `json:"next_file_index"`
	PendingZip    string   `json:"pending_zip,omitempty"`
	PartSerial    int      `json:"part_serial"`
	Done          bool     `json:"done"`
}

type RemoteStatus struct {
	TotalFiles   int    `json:"totalFiles"`
	PackedFiles  int    `json:"packedFiles"`
	PendingZip   string `json:"pendingZip"`
	Done         bool   `json:"done"`
	NextFileIndex int   `json:"nextFileIndex"`
}

type JobStatus struct {
	Running      bool   `json:"running"`
	Phase        string `json:"phase"`
	CurrentPart  string `json:"currentPart"`
	LocalFile    string `json:"localFile"`
	TotalFiles   int    `json:"totalFiles"`
	PackedFiles  int    `json:"packedFiles"`
	Done         bool   `json:"done"`
	LastError    string `json:"lastError"`
	RemoteInited bool   `json:"remoteInited"`
	PendingZip   string `json:"pendingZip,omitempty"`
	PrefetchZip  string `json:"prefetchZip,omitempty"`
	RemoteHint   string `json:"remoteHint,omitempty"`
	MaxFileBytes       int64   `json:"maxFileBytes"`
	OversizedFileCount int     `json:"oversizedFileCount"`
	TimingStartedAtMs        int64   `json:"timingStartedAtMs,omitempty"`
	TimingPackedFilesAtStart int     `json:"timingPackedFilesAtStart,omitempty"`
	TimingEstimatedTotalMs   int64   `json:"timingEstimatedTotalMs,omitempty"`
	DownloadBytesDone        int64   `json:"downloadBytesDone,omitempty"`
	DownloadBytesTotal       int64   `json:"downloadBytesTotal,omitempty"`
	DownloadSpeedBps         float64 `json:"downloadSpeedBps,omitempty"`
}
