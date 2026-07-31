package service

import (
	"strings"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/remote"
	"z-server-backup-tools/backend/util"
	"z-server-backup-tools/backend/zipbak"
)

func queryRemoteOversized(cfg model.BackupConfig) ([]zipbak.OversizedFile, int64, error) {
	cli, err := remote.Dial(cfg)
	if err != nil {
		return nil, 0, err
	}
	defer cli.Close()
	state := util.NormalizeRemotePathForOS(cfg.RemoteState, cfg.OSType)
	maxGB := formatMaxGBFlag(cfg)
	out, err := cli.RunRemote("oversized", "--state", state, "--max-gb", maxGB)
	if err != nil {
		return nil, 0, err
	}
	items, err := zipbak.ParseOversizedJSON(out)
	if err != nil {
		return nil, 0, err
	}
	maxB := cfg.MaxPartBytes()
	return items, maxB, nil
}

func appendOversizedWarnings(log func(string), cfg model.BackupConfig) {
	items, maxB, err := queryRemoteOversized(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "unknown") || strings.Contains(err.Error(), "用法") {
			return
		}
		log("检查超大文件失败: " + err.Error())
		return
	}
	for _, line := range zipbak.OversizedWarningLines(maxB, items) {
		log(line)
	}
}
