package main

import (
	"embed"
	"github.com/gailsapp/gails/pkg/application"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "File API Demo",
		Description: "File reading and handling",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "File API Demo",
		Width:  900,
		Height: 700,
		URL:    "/",
	})
	app.Run()
}
