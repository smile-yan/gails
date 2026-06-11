//go:build windows && (!production || devtools)

package application

import "github.com/gailsapp/gails/pkg/webview2"

func (w *windowsWebviewWindow) openDevTools() {
	w.chromium.OpenDevToolsWindow()
}

func (w *windowsWebviewWindow) enableDevTools(settings *webview2.Settings) {
	err := settings.PutAreDevToolsEnabled(true)
	if err != nil {
		globalApplication.handleFatalError(err)
	}
}
