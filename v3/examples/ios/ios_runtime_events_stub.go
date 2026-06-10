//go:build !ios

package main

import "github.com/gailsapp/gails/v3/pkg/application"

// Non-iOS: no-op so examples build on other platforms
func registerIOSRuntimeEventHandlers(app *application.App) {}
