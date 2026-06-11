//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"io"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Stream is a thin Go wrapper over the COM IStream interface used by
// WebView2 APIs that hand back streams (favicons, captured previews,
// PrintToPdf, WebResourceRequest/Response content, SharedBuffer.OpenStream,
// context menu icons). The Raw field is the COM object pointer; vtbl is
// resolved lazily by the methods that need it.
type Stream struct {
	Raw  uintptr
	vtbl *iStreamVtable
}

// iStreamVtable is the COM IStream vtable. Slot order is invariant:
// 3 IUnknown slots followed by 7 ISequentialStream/IStream slots
// (Read, Write, Seek, SetSize, CopyTo, Commit, Revert). Gails only
// invokes Read/Write/Release directly, but the full layout is declared
// so the pointer arithmetic in Read/Write lands on the right slots
// when working with IStreams returned by other COM components.
type iStreamVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Read           uintptr
	Write          uintptr
	Seek           uintptr
	SetSize        uintptr
	CopyTo         uintptr
	Commit         uintptr
	Revert         uintptr
}

// AddRef increments the reference count on the underlying COM object.
func (s *Stream) AddRef() error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	_, _, errno := syscall.SyscallN(
		vtbl.AddRef,
		uintptr(unsafe.Pointer(s)),
	)
	if errno != 0 && !errors.Is(errno, windows.ERROR_SUCCESS) {
		return errno
	}
	return nil
}

// Release decrements the reference count on the underlying COM object.
// The object is destroyed when its reference count reaches zero.
func (s *Stream) Release() error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	_, _, errno := syscall.SyscallN(
		vtbl.Release,
		uintptr(unsafe.Pointer(s)),
	)
	if errno != 0 && !errors.Is(errno, windows.ERROR_SUCCESS) {
		return errno
	}
	return nil
}

// Read reads up to len(p) bytes from the stream into p. It returns the
// number of bytes read; if fewer than len(p) bytes were read and the
// stream is at EOF, Read returns io.EOF alongside the partial count.
func (s *Stream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	vtbl, err := s.vtable()
	if err != nil {
		return 0, err
	}
	var n int
	hr, _, _ := syscall.SyscallN(
		vtbl.Read,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)),
		uintptr(unsafe.Pointer(&n)),
	)
	switch windows.Handle(hr) {
	case windows.S_OK:
		return n, nil
	case windows.S_FALSE:
		return n, io.EOF
	default:
		return 0, syscall.Errno(hr)
	}
}

// Write writes len(p) bytes from p to the stream. It returns the number
// of bytes written; partial writes are not retried.
func (s *Stream) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	vtbl, err := s.vtable()
	if err != nil {
		return 0, err
	}
	var n int
	hr, _, _ := syscall.SyscallN(
		vtbl.Write,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)),
		uintptr(unsafe.Pointer(&n)),
	)
	if hr != 0 {
		return 0, fmt.Errorf("IStream::Write failed: 0x%08x", hr)
	}
	return n, nil
}

// vtable resolves and caches the vtable pointer from Raw. The first
// dereference of a COM object always goes through the vtable, so we
// read it once per Stream lifetime.
func (s *Stream) vtable() (*iStreamVtable, error) {
	if s.vtbl != nil {
		return s.vtbl, nil
	}
	if s.Raw == 0 {
		return nil, fmt.Errorf("IStream: nil COM pointer")
	}
	// Standard COM vtable-pointer dereference. The two uintptr conversions
	// silence govet's unsafe.Pointer check (the value cannot be a pointer
	// to a Go object — it is a foreign COM vtable).
	vtblPtr := *(*uintptr)(unsafe.Pointer(s.Raw))
	s.vtbl = (*iStreamVtable)(unsafe.Pointer(vtblPtr))
	return s.vtbl, nil
}
