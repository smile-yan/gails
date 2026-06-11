//go:build windows

// Package w32helper is a thin port of upstream
// github.com/wailsapp/wails/webview2/internal/w32, providing a single
// syscall.Proc wrapper for Ole32CoInitializeEx.
package w32helper

import "syscall"

var (
	modole32              = syscall.NewLazyDLL("ole32.dll")
	Ole32CoInitializeEx   = modole32.NewProc("CoInitializeEx")
	Ole32CoUninitialize   = modole32.NewProc("CoUninitialize")
	Ole32CoCreateInstance = modole32.NewProc("CoCreateInstance")
)

const (
	COINIT_APARTMENTTHREADED = 0x2
)
