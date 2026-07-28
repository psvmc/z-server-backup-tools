package zipbak

import "fmt"
type Status struct {
	TotalFiles     int    `json:"totalFiles"`
	PackedFiles    int    `json:"packedFiles"`
	PendingZip     string `json:"pendingZip"`
	PrefetchZip    string `json:"prefetchZip,omitempty"`
	Done           bool   `json:"done"`
	NextFileIndex  int    `json:"nextFileIndex"`
	MaxFileBytes       int64  `json:"maxFileBytes"`
	OversizedFileCount int    `json:"oversizedFileCount"`
}

func (st *State) Status() Status {
	return Status{
		TotalFiles:    len(st.Files),
		PackedFiles:   st.NextFileIndex,
		PendingZip:    st.PendingZip,
		Done:          st.Done,
		NextFileIndex: st.NextFileIndex,
	}
}

func ValidateStatePaths(sourceDir, stagingDir string) error {
	if sourceDir == "" || stagingDir == "" {
		return fmt.Errorf("state 缺少 source_dir 或 staging_dir")
	}
	return nil
}
