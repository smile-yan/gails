//go:build windows

package main

import "github.com/gailsapp/gails/pkg/w32"

func init() {
	getExStyle = func() int {
		return w32.WS_EX_TOOLWINDOW | w32.WS_EX_NOREDIRECTIONBITMAP | w32.WS_EX_TOPMOST
	}
}
