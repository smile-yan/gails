//go:build !ios

package main

import "github.com/gailsapp/gails/pkg/application"

// Non-iOS: no-op so examples build on other platforms
func registerIOSRuntimeEventHandlers(app *application.App) {}
