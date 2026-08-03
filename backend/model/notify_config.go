package model

import "strings"

// NotifyConfig is the email_config block persisted in config.json (email/SMTP only).
type NotifyConfig struct {
	NotifyEmail  string `json:"notify_email,omitempty"`
	SmtpHost     string `json:"smtp_host,omitempty"`
	SmtpPort     int    `json:"smtp_port,omitempty"`
	SmtpUser     string `json:"smtp_user,omitempty"`
	SmtpPassword string `json:"smtp_password,omitempty"`
}

func NotifyConfigFrom(c BackupConfig) NotifyConfig {
	n := c.NotifyOnly()
	return NotifyConfig{
		NotifyEmail:  n.NotifyEmail,
		SmtpHost:     n.SmtpHost,
		SmtpPort:     n.SmtpPort,
		SmtpUser:     n.SmtpUser,
		SmtpPassword: n.SmtpPassword,
	}
}

func (n NotifyConfig) BackupConfig() BackupConfig {
	return BackupConfig{
		NotifyEmail:  strings.TrimSpace(n.NotifyEmail),
		SmtpHost:     strings.TrimSpace(n.SmtpHost),
		SmtpPort:     n.SmtpPort,
		SmtpUser:     strings.TrimSpace(n.SmtpUser),
		SmtpPassword: n.SmtpPassword,
	}
}
