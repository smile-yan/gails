//go:build windows

// C-side callbacks for the generic ComObject machinery. These
// functions are the entries in the IUnknown vTable. They run on
// COM's calling thread and must:
//
//   - Be callable through windows.NewCallback (so the function
//     signature must be a supported uintptr-returning shape).
//   - Hold no Go locks while calling into other Go code that
//     might recurse through the same callback (e.g. via
//     QueryInterface).
//
// The bodies mirror upstream
// github.com/wailsapp/wails/webview2/pkg/combridge/iunknown.go.

package bridge

import "golang.org/x/sys/windows"

func iUnknownQueryInterface(this uintptr, refiid *windows.GUID, ppvObject *uintptr) uintptr {
	if refiid == nil || ppvObject == nil {
		return uintptr(windows.E_INVALIDARG)
	}

	comIfcePointersL.RLock()
	obj := comIfcePointers[this]
	comIfcePointersL.RUnlock()

	ref := obj.queryInterface(refiid.String(), true)
	if ref != 0 {
		*ppvObject = ref
		return windows.NO_ERROR
	}

	*ppvObject = 0
	return uintptr(windows.E_NOINTERFACE)
}

func iUnknownAddRef(this uintptr) uintptr {
	comIfcePointersL.RLock()
	obj := comIfcePointers[this]
	comIfcePointersL.RUnlock()

	return uintptr(obj.addRef())
}

func iUnknownRelease(this uintptr) uintptr {
	comIfcePointersL.RLock()
	obj := comIfcePointers[this]
	comIfcePointersL.RUnlock()

	return uintptr(obj.release())
}
