//go:build !ios

package application

import "github.com/gailsapp/gails/v3/internal/signal"

// platformSignalHandler holds the signal handler for desktop platforms
type platformSignalHandler struct {
	signalHandler *signal.SignalHandler
}