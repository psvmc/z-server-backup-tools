package zipbak

type State struct {
	SourceDir     string   `json:"source_dir"`
	StagingDir    string   `json:"staging_dir"`
	MaxPartBytes  int64    `json:"max_part_bytes"`
	Files         []string `json:"files"`
	NextFileIndex int      `json:"next_file_index"`
	PendingZip    string   `json:"pending_zip,omitempty"`
	PartSerial    int      `json:"part_serial"`
	Done          bool     `json:"done"`
}
