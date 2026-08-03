package main

import (
	"embed"
	"log"

	"z-server-backup-tools/backend/service"
	"z-server-backup-tools/backend/update"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        AppName,
		Description: "远程 Windows 分卷 zip 流水线备份客户端",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.RegisterService(application.NewService(service.NewBackupService(app)))
	app.RegisterService(application.NewService(service.NewSingleFileBackupService(app)))

	if err := update.InitUpdater(app, AppVersion); err != nil {
		log.Fatal(err)
	}
	app.RegisterService(application.NewService(service.NewUpdateService(app, AppVersion)))

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  AppTitle(),
		Width:  1040,
		Height: 720,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(238, 246, 252),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
