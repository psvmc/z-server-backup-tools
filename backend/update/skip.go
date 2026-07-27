package update

import (
	"z-server-backup-tools/backend/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func applySkippedVersion(app *application.App) {
	store, err := config.NewStore()
	if err != nil {
		return
	}
	if version := store.GetSkippedUpdateVersion(); version != "" {
		app.Updater.SkipVersion(version)
	}
}
