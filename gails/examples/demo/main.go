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
		Name:        "Hello World",
		Description: "A minimal Hello World demo",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Hello World",
		Width:            600,
		Height:           400,
		URL:              "/",
		DevToolsEnabled:  true,
		BackgroundColour: application.NewRGB(30, 30, 35),
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
