package main

import (
	"embed"
	"log"

	"github.com/gailsapp/gails/pkg/application"
)

//go:embed assets/*
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Services: []application.Service{
			application.NewService(&Service{}),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		URL:             "/",
		DevToolsEnabled: true,
	})

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}

}
