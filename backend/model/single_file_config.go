package model

type SingleFileConfig struct {
	RemoteFile string `json:"remote_file"`
	LocalDir   string `json:"local_dir"`
}
