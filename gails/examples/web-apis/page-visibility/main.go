package main

import (
	"embed"
	"github.com/gailsapp/gails/pkg/application"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "Page Visibility Demo",
		Description: "Tab visibility detection",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Page Visibility Demo",
		Width:  900,
		Height: 700,
		URL:    "/",
	})
	app.Run()
}
