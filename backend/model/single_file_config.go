package model

type SingleFileConfig struct {
	ServerID   string `json:"server_id,omitempty"`
	RemoteFile string `json:"remote_file"`
	LocalDir   string `json:"local_dir"`
}
