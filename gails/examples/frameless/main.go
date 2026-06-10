package main

import (
	"embed"

	"log"

	"github.com/gailsapp/gails/pkg/application"
)

//go:embed assets
var assets embed.FS

func main() {

	app := application.New(application.Options{
		Name:        "Frameless Demo",
		Description: "A demo of frameless windows",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Frameless: true,
	})

	err := app.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
}
