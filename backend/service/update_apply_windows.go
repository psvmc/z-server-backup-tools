//go:build windows

package service

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"z-server-backup-tools/backend/update"
)

func (s *UpdateService) applyPlatformUpdate(ctx context.Context) error {
	if err := s.app.Updater.DownloadAndInstall(ctx); err != nil {
		return fmt.Errorf("???????: %w", err)
	}

	installerPath := s.app.Updater.DownloadedPath()
	if installerPath == "" {
		return fmt.Errorf("??????????")
	}

	if err := runWindowsInstaller(installerPath); err != nil {
		return err
	}

	s.app.Quit()
	return nil
}

func runWindowsInstaller(installerPath string) error {
	if filepath.Ext(installerPath) != ".exe" {
		return fmt.Errorf("?????????: %s", installerPath)
	}

	cmd := exec.Command(installerPath)
	update.ApplyVisibleDetachAttrs(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("????????: %w", err)
	}
	return nil
}
