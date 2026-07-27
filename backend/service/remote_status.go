package service

import (
	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/remote"
	"z-server-backup-tools/backend/util"
	"z-server-backup-tools/backend/zipbak"
)

func queryRemoteStatus(cfg model.BackupConfig) (zipbak.Status, error) {
	cli, err := remote.Dial(cfg)
	if err != nil {
		return zipbak.Status{}, err
	}
	defer cli.Close()
	state := util.NormalizeRemotePath(cfg.RemoteState)
	maxGB := formatMaxGBFlag(cfg)
	out, err := cli.RunRemote("status", "--state", state, "--max-gb", maxGB)
	if err != nil {
		return zipbak.Status{}, err
	}
	return zipbak.ParseStatusJSON(out)
}

func (s *BackupService) queryRemoteStatusLocked(cfg model.BackupConfig) (zipbak.Status, error) {
	s.remoteQueryMu.Lock()
	defer s.remoteQueryMu.Unlock()
	return queryRemoteStatus(cfg)
}

func (s *BackupService) applyRemoteStatus(st zipbak.Status, markInited bool) {
	s.status.TotalFiles = st.TotalFiles
	s.status.PackedFiles = st.PackedFiles
	s.status.Done = st.Done
	s.status.PendingZip = st.PendingZip
	s.status.MaxFileBytes = st.MaxFileBytes
	s.status.OversizedFileCount = st.OversizedFileCount
	if markInited || st.TotalFiles > 0 {
		s.status.RemoteInited = true
	}
	s.status.LastError = ""
}
