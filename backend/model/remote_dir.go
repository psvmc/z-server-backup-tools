package model

// RemoteDirEntry is one selectable directory on the remote Windows host (via SFTP).
type RemoteDirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// RemoteDirListing is returned when browsing remote folders in the UI.
type RemoteDirListing struct {
	CurrentPath string           `json:"current_path"`
	ParentPath  string           `json:"parent_path"`
	Entries     []RemoteDirEntry `json:"entries"`
}
