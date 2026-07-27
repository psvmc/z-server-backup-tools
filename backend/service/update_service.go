package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"z-server-backup-tools/backend/config"
	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/update"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	updateCheckTimeout    = 30 * time.Second
	updateDownloadTimeout = 10 * time.Minute
)

type UpdateService struct {
	app            *application.App
	currentVersion string
	store          *config.Store
}

func NewUpdateService(app *application.App, currentVersion string) *UpdateService {
	store, _ := config.Default()
	return &UpdateService{app: app, currentVersion: currentVersion, store: store}
}

func (s *UpdateService) GetCurrentVersion() string {
	return s.currentVersion
}

func (s *UpdateService) CheckForUpdate() (model.UpdateCheckResult, error) {
	result := model.UpdateCheckResult{
		CurrentVersion: s.currentVersion,
		Enabled:        update.Enabled,
	}
	if !update.Enabled {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	rel, err := s.app.Updater.Check(ctx)
	if err != nil {
		return result, fmt.Errorf("??????: %w", err)
	}
	if rel == nil {
		return result, nil
	}

	result.Available = true
	result.LatestVersion = rel.Version
	result.ReleaseName = rel.Name
	result.Notes = rel.Notes
	if rel.Metadata != nil {
		if url, ok := rel.Metadata["github.release.htmlURL"].(string); ok {
			result.ReleaseURL = url
		}
	}
	return result, nil
}

func (s *UpdateService) ApplyUpdate() error {
	if !update.Enabled {
		return fmt.Errorf("???????????")
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateDownloadTimeout)
	defer cancel()

	return s.applyPlatformUpdate(ctx)
}

func (s *UpdateService) SkipUpdateVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("???????")
	}
	if !update.Enabled {
		return fmt.Errorf("???????????")
	}

	s.app.Updater.SkipVersion(version)
	if s.store == nil {
		return nil
	}
	return s.store.SetSkippedUpdateVersion(version)
}
