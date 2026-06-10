//go:build windows

package application

import (
	"github.com/gailsapp/gails/v3/pkg/w32"
	"sync"
)

type windowsClipboard struct {
	lock sync.RWMutex
}

func (m *windowsClipboard) setText(text string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return w32.SetClipboardText(text) == nil
}

func (m *windowsClipboard) text() (string, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	text, err := w32.GetClipboardText()
	return text, err == nil
}

func newClipboardImpl() *windowsClipboard {
	return &windowsClipboard{}
}
