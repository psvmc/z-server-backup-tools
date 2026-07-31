package model

type Server struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Host             string  `json:"host"`
	Port             int     `json:"port"`
	User             string  `json:"user"`
	Password         string  `json:"password"`
	SupportMultiFile bool    `json:"support_multi_file"`
	RemoteAppDir     string  `json:"remote_app_dir,omitempty"`
	MaxPartGB        float64 `json:"max_part_gb,omitempty"`
}

func (s Server) ApplyTo(base BackupConfig) BackupConfig {
	out := base
	out.Host = s.Host
	out.Port = s.Port
	out.User = s.User
	out.Password = s.Password
	out.RemoteAppDir = s.RemoteAppDir
	if s.MaxPartGB > 0 {
		out.MaxPartGB = s.MaxPartGB
	}
	return out
}
