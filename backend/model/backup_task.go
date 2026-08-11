package model

import "strings"

// BackupTask is one backup job (source path, local dir, zip prefix).
type BackupTask struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Kind           string `json:"kind,omitempty"` // "multi" | "single" | "folder_zip"; empty = multi
	ServerID       string `json:"server_id,omitempty"`
	RemoteSource   string `json:"remote_source"`
	LocalDir       string   `json:"local_dir"`
	PartNamePrefix string   `json:"part_name_prefix,omitempty"`
	IgnorePatterns []string `json:"ignore_patterns,omitempty"` // folder_zip: filename/path regex excludes
}

func (t BackupTask) DisplayName() string {
	if n := strings.TrimSpace(t.Name); n != "" {
		return n
	}
	if s := strings.TrimSpace(t.RemoteSource); s != "" {
		return s
	}
	return t.ID
}

func (t BackupTask) NormalizedKind() string {
	switch strings.ToLower(strings.TrimSpace(t.Kind)) {
	case "single":
		return "single"
	case "folder_zip":
		return "folder_zip"
	default:
		return "multi"
	}
}

// MergeInto copies task fields into a base connection config.
func (t BackupTask) MergeInto(base BackupConfig) BackupConfig {
	out := base
	out.TaskID = strings.TrimSpace(t.ID)
	out.RemoteSource = strings.TrimSpace(t.RemoteSource)
	out.LocalDir = strings.TrimSpace(t.LocalDir)
	out.PartNamePrefix = strings.TrimSpace(t.PartNamePrefix)
	return out
}
