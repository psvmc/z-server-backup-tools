//go:build darwin

package service

import (
	"context"
	"fmt"

	"z-server-backup-tools/backend/update"
)

func (s *UpdateService) applyPlatformUpdate(ctx context.Context) error {
	if err := s.app.Updater.DownloadAndInstall(ctx); err != nil {
		return fmt.Errorf("?????????: %w", err)
	}

	staged := s.app.Updater.DownloadedPath()
	if staged == "" {
		return fmt.Errorf("??????????")
	}

	if err := update.ApplyStagedUpdate(staged); err != nil {
		return fmt.Errorf("??????: %w", err)
	}

	s.app.Quit()
	return nil
}
