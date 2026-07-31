package model

import "strings"

const (
	OSTypeWindows = "windows"
	OSTypeLinux   = "linux"
)

type Server struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Host             string  `json:"host"`
	Port             int     `json:"port"`
	User             string  `json:"user"`
	Password         string  `json:"password"`
	OSType           string  `json:"os_type"`
	SupportMultiFile bool    `json:"support_multi_file"`
	RemoteAppDir     string  `json:"remote_app_dir,omitempty"`
	MaxPartGB        float64 `json:"max_part_gb,omitempty"`
}

func NormalizeOSType(osType string) string {
	switch strings.ToLower(strings.TrimSpace(osType)) {
	case OSTypeLinux:
		return OSTypeLinux
	default:
		return OSTypeWindows
	}
}

func IsLinuxOS(osType string) bool {
	return NormalizeOSType(osType) == OSTypeLinux
}

func (s Server) ApplyTo(base BackupConfig) BackupConfig {
	out := base
	out.Host = s.Host
	out.Port = s.Port
	out.User = s.User
	out.Password = s.Password
	out.OSType = NormalizeOSType(s.OSType)
	out.RemoteAppDir = s.RemoteAppDir
	if s.MaxPartGB > 0 {
		out.MaxPartGB = s.MaxPartGB
	}
	return out
}
