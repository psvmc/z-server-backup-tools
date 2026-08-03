package model

// RemoteDirEntry is one selectable directory or file on the remote host (via SFTP).
type RemoteDirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// RemoteDirListing is returned when browsing remote folders/files in the UI.
type RemoteDirListing struct {
	CurrentPath string           `json:"current_path"`
	ParentPath  string           `json:"parent_path"`
	Entries     []RemoteDirEntry `json:"entries"`
}
