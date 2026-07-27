package update

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func InitUpdater(app *application.App, currentVersion string) error {
	if !Enabled {
		return nil
	}

	provider, err := github.New(github.Config{
		Repository:   GitHubRepository,
		AssetMatcher: AssetMatcher,
		HTTPClient:   updateHTTPClient(),
	})
	if err != nil {
		return fmt.Errorf("init github update provider: %w", err)
	}

	if err := app.Updater.Init(updater.Config{
		CurrentVersion: currentVersion,
		Providers:      []updater.Provider{provider},
		Window:         updater.WindowNone,
	}); err != nil {
		return err
	}

	applySkippedVersion(app)
	return nil
}
