package main

import (
	"embed"
	"github.com/gailsapp/gails/pkg/application"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "Mutation Observer Demo",
		Description: "DOM change observation",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Mutation Observer Demo",
		Width:  900,
		Height: 700,
		URL:    "/",
	})
	app.Run()
}
