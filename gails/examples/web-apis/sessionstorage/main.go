package main

import (
	"embed"
	"github.com/gailsapp/gails/pkg/application"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "Session Storage Demo",
		Description: "sessionStorage API demonstration",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Session Storage Demo",
		Width:  900,
		Height: 700,
		URL:    "/",
	})
	app.Run()
}
