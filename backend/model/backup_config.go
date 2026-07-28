package model

import (
	"strings"

	"z-server-backup-tools/backend/util"
)

// BackupConfig mirrors zipbak-cli.json for the desktop client.
type BackupConfig struct {
	Host           string  `json:"host"`
	Port           int     `json:"port"`
	User           string  `json:"user"`
	Password       string  `json:"password"`
	RemoteAppDir   string  `json:"remote_app_dir"`
	RemoteSrv      string  `json:"remote_srv,omitempty"`
	RemoteState    string  `json:"remote_state,omitempty"`
	RemoteSource   string  `json:"remote_source"`
	RemoteStaging  string  `json:"remote_staging,omitempty"`
	LocalDir       string  `json:"local_dir"`
	MaxPartGB      float64 `json:"max_part_gb"`
	KnownHostsFile string  `json:"known_hosts_file,omitempty"`
}

func (c BackupConfig) MaxPartBytes() int64 {
	if c.MaxPartGB <= 0 {
		return 2 << 30
	}
	return int64(c.MaxPartGB * (1 << 30))
}

// Resolved fills exe / state / staging from RemoteAppDir when set.
func (c BackupConfig) Resolved() BackupConfig {
	out := c
	app := strings.TrimSpace(c.RemoteAppDir)
	if app == "" {
		return out
	}
	base := util.NormalizeRemotePath(app)
	out.RemoteSrv = util.JoinRemote(base, "zipbak-srv.exe")
	out.RemoteState = util.JoinRemote(base, "data", "state.db")
	out.RemoteStaging = util.JoinRemote(base, "staging")
	return out
}
