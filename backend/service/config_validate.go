package service

import (
	"fmt"
	"strings"

	"z-server-backup-tools/backend/model"
)

func prepareSSH(cfg model.BackupConfig) (model.BackupConfig, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	if cfg.Host == "" {
		return cfg, fmt.Errorf("SSH 主机不能为空（填写后建议点「保存配置」）")
	}
	if cfg.User == "" {
		return cfg, fmt.Errorf("SSH 用户名不能为空")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	return cfg.Resolved(), nil
}

func prepareBackupJob(cfg model.BackupConfig) (model.BackupConfig, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	cfg.RemoteAppDir = strings.TrimSpace(cfg.RemoteAppDir)
	cfg.RemoteSource = strings.TrimSpace(cfg.RemoteSource)
	cfg.LocalDir = strings.TrimSpace(cfg.LocalDir)
	if cfg.Host == "" || cfg.User == "" {
		return cfg, fmt.Errorf("主机与用户名不能为空")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.MaxPartGB <= 0 {
		cfg.MaxPartGB = 2
	}
	if cfg.RemoteAppDir == "" {
		return cfg, fmt.Errorf("远程应用目录不能为空")
	}
	if cfg.RemoteSource == "" {
		return cfg, fmt.Errorf("远程源目录不能为空")
	}
	if cfg.LocalDir == "" {
		return cfg, fmt.Errorf("本机保存目录不能为空")
	}
	cfg.RemoteSrv = ""
	cfg.RemoteState = ""
	cfg.RemoteStaging = ""
	return cfg.Resolved(), nil
}
