package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"z-server-backup-tools/backend/config"
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
		return cfg, fmt.Errorf("远程源目录不能为空，请先添加并选择备份任务")
	}
	if cfg.LocalDir == "" {
		return cfg, fmt.Errorf("本机保存目录不能为空，请先添加并选择备份任务")
	}
	if strings.TrimSpace(cfg.TaskID) == "" {
		return cfg, fmt.Errorf("请先选择备份任务")
	}
	cfg.RemoteSrv = ""
	cfg.RemoteState = ""
	cfg.RemoteStaging = ""
	return cfg.Resolved(), nil
}

func mergeServerTaskNotify(notify model.BackupConfig, srv model.Server, task model.BackupTask) model.BackupConfig {
	return task.MergeInto(srv.ApplyTo(notify)).Resolved()
}

func findServer(servers []model.Server, id string) (model.Server, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.Server{}, fmt.Errorf("请选择服务器")
	}
	for _, srv := range servers {
		if srv.ID == id {
			return srv, nil
		}
	}
	return model.Server{}, fmt.Errorf("服务器不存在: %s", id)
}

func lookupServer(store *config.Store, id string) (model.Server, error) {
	if store == nil {
		return model.Server{}, fmt.Errorf("配置存储不可用")
	}
	return findServer(store.GetServers(), id)
}

func normalizeServer(srv model.Server) (model.Server, error) {
	srv.ID = strings.TrimSpace(srv.ID)
	srv.Name = strings.TrimSpace(srv.Name)
	srv.Host = strings.TrimSpace(srv.Host)
	srv.User = strings.TrimSpace(srv.User)
	srv.RemoteAppDir = strings.TrimSpace(srv.RemoteAppDir)
	srv.OSType = model.NormalizeOSType(srv.OSType)
	if srv.Name == "" {
		return model.Server{}, fmt.Errorf("服务器名称不能为空")
	}
	if srv.Host == "" {
		return model.Server{}, fmt.Errorf("SSH 主机不能为空")
	}
	if srv.User == "" {
		return model.Server{}, fmt.Errorf("SSH 用户名不能为空")
	}
	if srv.Port == 0 {
		srv.Port = 22
	}
	if srv.SupportMultiFile {
		if srv.RemoteAppDir == "" {
			return model.Server{}, fmt.Errorf("远程应用目录不能为空")
		}
		if srv.MaxPartGB <= 0 {
			return model.Server{}, fmt.Errorf("分卷上限必须大于 0")
		}
	} else {
		srv.RemoteAppDir = ""
		srv.MaxPartGB = 0
	}
	if srv.ID == "" {
		srv.ID = newEntityID()
	}
	return srv, nil
}

func newEntityID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", len(b))
	}
	return hex.EncodeToString(b[:])
}
