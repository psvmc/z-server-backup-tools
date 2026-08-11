package model

type FolderZipConfig struct {
	ServerID       string   `json:"server_id,omitempty"`
	RemoteFolder   string   `json:"remote_folder"`
	LocalDir       string   `json:"local_dir"`
	IgnorePatterns []string `json:"ignore_patterns,omitempty"`
}
