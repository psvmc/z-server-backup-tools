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
	OSType         string  `json:"os_type,omitempty"`
	RemoteAppDir   string  `json:"remote_app_dir"`
	RemoteSrv      string  `json:"remote_srv,omitempty"`
	RemoteState    string  `json:"remote_state,omitempty"`
	RemoteSource   string  `json:"remote_source"`
	RemoteStaging  string  `json:"remote_staging,omitempty"`
	LocalDir       string  `json:"local_dir"`
	MaxPartGB      float64 `json:"max_part_gb"`
	PartNamePrefix string  `json:"part_name_prefix,omitempty"`
	TaskID         string  `json:"task_id,omitempty"`
	NotifyEmail    string  `json:"notify_email,omitempty"`
	SmtpHost       string  `json:"smtp_host,omitempty"`
	SmtpPort       int     `json:"smtp_port,omitempty"`
	SmtpUser       string  `json:"smtp_user,omitempty"`
	SmtpPassword   string  `json:"smtp_password,omitempty"`
	KnownHostsFile string  `json:"known_hosts_file,omitempty"`
}

// NotifyOnly returns email/SMTP fields for config.json email_config persistence.
func (c BackupConfig) NotifyOnly() BackupConfig {
	return BackupConfig{
		NotifyEmail:  strings.TrimSpace(c.NotifyEmail),
		SmtpHost:     strings.TrimSpace(c.SmtpHost),
		SmtpPort:     c.SmtpPort,
		SmtpUser:     strings.TrimSpace(c.SmtpUser),
		SmtpPassword: c.SmtpPassword,
	}
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
	osType := NormalizeOSType(c.OSType)
	base := util.NormalizeRemotePathForOS(app, osType)
	exe := "zipbak-srv.exe"
	if IsLinuxOS(osType) {
		exe = "zipbak-srv"
	}
	out.RemoteSrv = util.JoinRemoteForOS(osType, base, exe)
	suffix := taskPathSuffix(c.TaskID)
	out.RemoteState = util.JoinRemoteForOS(osType, base, "data", "state"+suffix+".db")
	out.RemoteStaging = util.JoinRemoteForOS(osType, base, "staging"+suffix)
	return out
}

func taskPathSuffix(taskID string) string {
	id := strings.TrimSpace(taskID)
	if id == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return ""
	}
	return "-" + s
}
