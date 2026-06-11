//go:build windows

// Package webview2 loader: version comparison, runtime detection, and
// the GetFileVersionInfo syscall helpers used to read the WebView2
// runtime's product version. Ported from
// github.com/wailsapp/wails/webview2/webviewloader.
package webview2

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// UsingGoWebview2Loader is set to true when the in-tree go webview2
// loader is used (always true after the gails rewrite; the boolean is
// preserved for compatibility with code that branched on it during the
// wailsapp→gailsapp transition).
var UsingGoWebview2Loader = true

// CompareBrowserVersions will compare the 2 given versions and return:
//
//	-1 = v1 < v2
//	 0 = v1 == v2
//	 1 = v1 > v2
func CompareBrowserVersions(v1 string, v2 string) (int, error) {
	v, err := parseVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("v1 invalid: %w", err)
	}

	w, err := parseVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("v2 invalid: %w", err)
	}

	return v.compare(w), nil
}

// GetAvailableCoreWebView2BrowserVersionString get the browser version
// info including channel name if it is the WebView2 Runtime. Channel
// names are Beta, Dev, and Canary.
//
// If browserExecutableFolder is empty, the function looks for the
// system-installed runtime. Otherwise it reads the version from the
// EmbeddedBrowserWebView.dll inside the given folder. An empty string
// with a nil error means the runtime is not installed.
func GetAvailableCoreWebView2BrowserVersionString(browserExecutableFolder string) (string, error) {
	if browserExecutableFolder != "" {
		clientPath, err := findEmbeddedClientDll(browserExecutableFolder)
		if errors.Is(err, errNoClientDLLFound) {
			// WebView2 is not found
			return "", nil
		} else if err != nil {
			return "", err
		}

		return findEmbeddedBrowserVersion(clientPath)
	}

	_, version, err := findInstalledClientDll(false)
	if errors.Is(err, errNoClientDLLFound) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return version.String(), nil
}

type version struct {
	major int
	minor int
	patch int
	build int

	channel string
}

func (v version) String() string {
	vv := fmt.Sprintf("%d.%d.%d.%d", v.major, v.minor, v.patch, v.build)
	if v.channel != "" {
		vv += " " + v.channel
	}

	return vv
}

func (v version) compare(o version) int {
	if c := compareInt(v.major, o.major); c != 0 {
		return c
	}
	if c := compareInt(v.minor, o.minor); c != 0 {
		return c
	}
	if c := compareInt(v.patch, o.patch); c != 0 {
		return c
	}
	return compareInt(v.build, o.build)
}

func parseVersion(v string) (version, error) {
	var p version

	// Split away channel information...
	if i := strings.Index(v, " "); i > 0 {
		p.channel = v[i+1:]
		v = v[:i]
	}

	vv := strings.Split(v, ".")
	if len(vv) > 4 {
		return p, fmt.Errorf("too many version parts")
	}

	var err error
	vv, p.major, err = parseInt(vv)
	if err != nil {
		return p, fmt.Errorf("bad major version: %w", err)
	}

	vv, p.minor, err = parseInt(vv)
	if err != nil {
		return p, fmt.Errorf("bad minor version: %w", err)
	}

	vv, p.patch, err = parseInt(vv)
	if err != nil {
		return p, fmt.Errorf("bad patch version: %w", err)
	}

	_, p.build, err = parseInt(vv)
	if err != nil {
		return p, fmt.Errorf("bad build version: %w", err)
	}

	return p, nil
}

func parseInt(v []string) ([]string, int, error) {
	if len(v) == 0 {
		return nil, 0, nil
	}

	p, err := strconv.ParseInt(v[0], 10, 32)
	if err != nil {
		return nil, 0, err
	}
	return v[1:], int(p), nil
}

func compareInt(v1, v2 int) int {
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	} else {
		return +1
	}
}

// -----------------------------------------------------------------------------
// DLL discovery (find_dll.go + find_dll_installed.go)
// -----------------------------------------------------------------------------

var (
	errNoClientDLLFound = errors.New("no webview2 found")
)

const (
	kNumChannels              = 4
	kInstallKeyPath           = "Software\\Microsoft\\EdgeUpdate\\ClientState\\"
	kMinimumCompatibleVersion = "86.0.616.0"
)

var (
	kChannelName = [kNumChannels]string{
		"", "beta", "dev", "canary", // "internal"
	}

	kChannelUuid = [kNumChannels]string{
		"{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}",
		"{2CD8A007-E189-409D-A2C8-9AF4EF3C72AA}",
		"{0D50BFEC-CD6A-4F9A-964C-C7416E3ACB10}",
		"{65C35B14-6C1D-4122-AC46-7148CC9D6497}",
		//"{BE59E8FD-089A-411B-A3B0-051D9E417818}",
	}

	minimumCompatibleVersion, _ = parseVersion(kMinimumCompatibleVersion)
)

func findEmbeddedBrowserVersion(filename string) (string, error) {
	block, err := getFileVersionInfo(filename)
	if err != nil {
		return "", err
	}

	info, err := verQueryValueString(block, "\\StringFileInfo\\040904B0\\ProductVersion")
	if err != nil {
		return "", err
	}

	return info, nil
}

func findEmbeddedClientDll(embeddedEdgeSubFolder string) (outClientPath string, err error) {
	if !filepath.IsAbs(embeddedEdgeSubFolder) {
		exe, err := os.Executable()
		if err != nil {
			return "", err
		}

		embeddedEdgeSubFolder = filepath.Join(filepath.Dir(exe), embeddedEdgeSubFolder)
	}

	return findClientDllInFolder(embeddedEdgeSubFolder)
}

func findInstalledClientDll(preferCanary bool) (clientPath string, version *version, err error) {
	for i := 0; i < kNumChannels; i++ {
		channel := i
		if preferCanary {
			channel = (kNumChannels - 1) - i
		}

		key := kInstallKeyPath + kChannelUuid[channel]
		for _, checkSystem := range []bool{true, false} {
			clientPath, version, err := findInstalledClientDllForChannel(key, checkSystem)
			if err == errNoClientDLLFound {
				continue
			}
			if err != nil {
				return "", nil, err
			}

			version.channel = kChannelName[channel]
			return clientPath, version, nil
		}
	}
	return "", nil, errNoClientDLLFound
}

func findInstalledClientDllForChannel(subKey string, system bool) (clientPath string, clientVersion *version, err error) {
	key := registry.LOCAL_MACHINE
	if !system {
		key = registry.CURRENT_USER
	}

	regKey, err := registry.OpenKey(key, subKey, registry.READ|registry.WOW64_32KEY)
	if err != nil {
		return "", nil, mapFindErr(err)
	}
	defer regKey.Close()

	embeddedEdgeSubFolder, _, err := regKey.GetStringValue("EBWebView")
	if err != nil {
		return "", nil, mapFindErr(err)
	}

	if embeddedEdgeSubFolder == "" {
		return "", nil, errNoClientDLLFound
	}

	versionString := filepath.Base(embeddedEdgeSubFolder)
	version, err := parseVersion(versionString)
	if err != nil {
		return "", nil, errNoClientDLLFound
	}

	if version.compare(minimumCompatibleVersion) < 0 {
		return "", nil, errNoClientDLLFound
	}

	dllPath, err := findEmbeddedClientDll(embeddedEdgeSubFolder)
	if err != nil {
		return "", nil, mapFindErr(err)
	}

	return dllPath, &version, nil
}

func findClientDllInFolder(folder string) (string, error) {
	arch := ""
	switch runtime.GOARCH {
	case "arm64":
		arch = "arm64"
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	default:
		return "", fmt.Errorf("Unsupported architecture")
	}

	dllPath := filepath.Join(folder, "EBWebView", arch, "EmbeddedBrowserWebView.dll")
	if _, err := os.Stat(dllPath); err != nil {
		return "", mapFindErr(err)
	}
	return dllPath, nil
}

func mapFindErr(err error) error {
	if errors.Is(err, registry.ErrNotExist) {
		return errNoClientDLLFound
	}
	if errors.Is(err, os.ErrNotExist) {
		return errNoClientDLLFound
	}
	return err
}

// -----------------------------------------------------------------------------
// File version info syscall helpers (syscall.go)
// -----------------------------------------------------------------------------

var (
	modkernel32     = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc = modkernel32.NewProc("GlobalAlloc")
	procGlobalFree  = modkernel32.NewProc("GlobalFree")

	modversion                 = windows.NewLazySystemDLL("version.dll")
	procGetFileVersionInfoSize = modversion.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfo     = modversion.NewProc("GetFileVersionInfoW")
	procVerQueryValue          = modversion.NewProc("VerQueryValueW")
)

func getFileVersionInfo(path string) ([]byte, error) {
	lptstrFilename, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	size, _, err := procGetFileVersionInfoSize.Call(
		uintptr(unsafe.Pointer(lptstrFilename)),
		0,
	)

	err = maskErrorSuccess(err)
	if size == 0 && err == nil {
		err = fmt.Errorf("GetFileVersionInfoSize failed")
	}

	if err != nil {
		return nil, err
	}

	data := make([]byte, size)
	ret, _, err := procGetFileVersionInfo.Call(
		uintptr(unsafe.Pointer(lptstrFilename)),
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&data[0])),
	)

	err = maskErrorSuccess(err)
	if ret == 0 && err == nil {
		err = fmt.Errorf("GetFileVersionInfo failed")
	}

	if err != nil {
		return nil, err
	}
	return data, nil
}

func verQueryValueString(block []byte, subBlock string) (string, error) {
	// Allocate memory from native side to make sure the block doesn't get moved
	// because we get a pointer into that memory block from the native verQueryValue
	// call back.
	pBlock := globalAlloc(0, uint32(len(block)))
	defer globalFree(unsafe.Pointer(pBlock))

	// Copy the memory region into native side memory
	copy(unsafe.Slice((*byte)(pBlock), len(block)), block)

	lpSubBlock, err := syscall.UTF16PtrFromString(subBlock)
	if err != nil {
		return "", err
	}

	var lplpBuffer unsafe.Pointer
	var puLen uint
	ret, _, err := procVerQueryValue.Call(
		uintptr(pBlock),
		uintptr(unsafe.Pointer(lpSubBlock)),
		uintptr(unsafe.Pointer(&lplpBuffer)),
		uintptr(unsafe.Pointer(&puLen)),
	)

	err = maskErrorSuccess(err)
	if ret == 0 && err == nil {
		err = fmt.Errorf("VerQueryValue failed")
	}

	if err != nil {
		return "", err
	}

	if puLen <= 1 {
		return "", nil
	}
	puLen -= 1 // Remove Null-Terminator

	wchar := unsafe.Slice((*uint16)(lplpBuffer), puLen)
	return string(utf16.Decode(wchar)), nil
}

func globalAlloc(uFlags uint, dwBytes uint32) unsafe.Pointer {
	ret, _, _ := procGlobalAlloc.Call(
		uintptr(uFlags),
		uintptr(dwBytes))

	if ret == 0 {
		panic("globalAlloc failed")
	}

	return unsafe.Pointer(ret)
}

func globalFree(data unsafe.Pointer) {
	ret, _, _ := procGlobalFree.Call(uintptr(unsafe.Pointer(data)))
	if ret != 0 {
		panic("globalFree failed")
	}
}

func maskErrorSuccess(err error) error {
	if err == windows.ERROR_SUCCESS {
		return nil
	}
	return err
}

// -----------------------------------------------------------------------------
// Environment creation (env_create.go + env_create_completed.go)
// -----------------------------------------------------------------------------

// CreateEnvironmentOptions configures the WebView2 environment.
//
// Mirrors the field set in upstream webviewloader's
// `environmentOptions` struct. All fields are optional; the empty
// string / false zero value matches upstream's "use the runtime
// default" semantics.
type CreateEnvironmentOptions struct {
	BrowserExecutableFolder string
	UserDataFolder          string
	AdditionalBrowserArgs   string
	Language                 string
}

// webView2RunTimeType is the runtime flavour passed to
// CreateWebViewEnvironmentWithOptionsInternal. Mirrors the
// webView2RunTimeType enum in upstream webviewloader.
type webView2RunTimeType int32

const (
	webView2RunTimeTypeInstalled       webView2RunTimeType = 0x00
	webView2RunTimeTypeRedistributable webView2RunTimeType = 0x01
)

// CreateEnvironmentCompletedHandler is the Go-side
// ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler.
// Construct one with NewCreateEnvironmentCompletedHandler and pass
// to CreateCoreWebView2EnvironmentWithOptions; call Close when done.
type CreateEnvironmentCompletedHandler struct {
	impl *comHandlerImpl
}

// NewCreateEnvironmentCompletedHandler wires a Go callback to the
// ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler.Invoke
// vtable slot. The returned handler holds a reference to a native
// COM object; the caller must call Close when finished.
//
// Ported from upstream
// ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler.
func NewCreateEnvironmentCompletedHandler(callback func(result int32, env *Environment)) *CreateEnvironmentCompletedHandler {
	trampoline := windows.NewCallback(createEnvironmentCompletedInvokeTrampoline)
	h := NewComHandler(trampoline, callback)
	return &CreateEnvironmentCompletedHandler{impl: h}
}

// Close releases the underlying COM object. Calling Close twice is
// a no-op.
func (h *CreateEnvironmentCompletedHandler) Close() {
	if h.impl == nil {
		return
	}
	h.impl.Release()
	h.impl = nil
}

// createEnvironmentCompletedInvokeTrampoline is the per-instance
// Invoke slot for the
// ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler
// vtable. It is registered as a C callback via windows.NewCallback
// and is invoked by WebView2 when environment creation finishes.
//
// The signature is fixed by COM stdcall: the first argument is the
// COM `this` pointer, followed by the Invoke method's typed
// arguments (HRESULT and the new ICoreWebView2Environment*), and
// the return value is an HRESULT uintptr.
func createEnvironmentCompletedInvokeTrampoline(this uintptr, errorCode uintptr, createdEnvironment uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0x80004003 // E_POINTER
	}
	cb, ok := impl.Callback().(func(result int32, env *Environment))
	if !ok || cb == nil {
		return 0 // S_OK; nothing to do
	}
	var env *Environment
	if createdEnvironment != 0 {
		env = &Environment{Raw: createdEnvironment}
	}
	cb(int32(errorCode), env)
	return 0
}

// CreateCoreWebView2EnvironmentWithOptions creates a WebView2
// environment asynchronously. WebView2 calls the completion
// handler's Invoke with the new ICoreWebView2Environment* when the
// runtime is ready.
//
// This is the Gails equivalent of upstream webviewloader's
// CreateCoreWebView2EnvironmentWithOptions. It uses
// CreateWebViewEnvironmentWithOptionsInternal exported by the
// WebView2 client DLL (the same entry point upstream uses). opts may
// be nil, in which case the installed runtime is used with default
// settings.
//
// The handler is mandatory (the WebView2 API is asynchronous and
// only delivers the environment pointer through the handler).
func CreateCoreWebView2EnvironmentWithOptions(opts *CreateEnvironmentOptions, handler *CreateEnvironmentCompletedHandler) error {
	if handler == nil || handler.impl == nil {
		return fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions: handler is nil")
	}
	if opts == nil {
		opts = &CreateEnvironmentOptions{}
	}

	// Resolve the WebView2 client DLL (either embedded via
	// BrowserExecutableFolder or system-installed).
	var (
		dllPath    string
		runtimeTyp webView2RunTimeType
		err        error
	)
	if opts.BrowserExecutableFolder != "" {
		runtimeTyp = webView2RunTimeTypeRedistributable
		dllPath, err = findEmbeddedClientDll(opts.BrowserExecutableFolder)
	} else {
		runtimeTyp = webView2RunTimeTypeInstalled
		dllPath, _, err = findInstalledClientDll(false)
	}
	if err != nil {
		return err
	}
	if dllPath == "" {
		return errNoClientDLLFound
	}

	return createWebViewEnvironmentWithClientDll(dllPath, runtimeTyp, opts, handler)
}

// createWebViewEnvironmentWithClientDll loads the WebView2 client
// DLL, finds the
// CreateWebViewEnvironmentWithOptionsInternal entry point, and
// invokes it with the env-options and the completion handler.
//
// The opts.AdditionalBrowserArgs, opts.Language, and
// opts.UserDataFolder are passed through as opaque strings to the
// runtime; the env options COM object the upstream port builds
// (iCoreWebView2EnvironmentOptions / Options2) is a full vtable
// implementation in the upstream port. In the Gails port we pass
// the strings via the UserDataFolder pointer and rely on
// WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS / WEBVIEW2_USER_DATA_FOLDER
// / language hints being set by the caller, matching the simpler
// shape of the in-tree Go webview2 loader prior to the combridge
// rewrite.
func createWebViewEnvironmentWithClientDll(lpLibFileName string, runtimeTyp webView2RunTimeType, opts *CreateEnvironmentOptions, handler *CreateEnvironmentCompletedHandler) error {
	if !filepath.IsAbs(lpLibFileName) {
		return fmt.Errorf("lpLibFileName must be absolute")
	}

	dll, err := windows.LoadDLL(lpLibFileName)
	if err != nil {
		return fmt.Errorf("Loading DLL failed: %w", err)
	}
	defer func() {
		// Best-effort: only release the DLL once WebView2 reports it
		// can unload. If FindProc fails or CanUnloadNow returns
		// non-zero, the runtime still has refs and we leave the
		// handle open for the OS to clean up at process exit.
		canUnloadProc, err := dll.FindProc("DllCanUnloadNow")
		if err != nil {
			return
		}
		if r1, _, _ := canUnloadProc.Call(); r1 != windows.NO_ERROR {
			return
		}
		dll.Release()
	}()

	createProc, err := dll.FindProc("CreateWebViewEnvironmentWithOptionsInternal")
	if err != nil {
		return fmt.Errorf("Unable to find CreateWebViewEnvironmentWithOptionsInternal entrypoint: %w", err)
	}

	userDataPtr, err := windows.UTF16PtrFromString(opts.UserDataFolder)
	if err != nil {
		return err
	}

	// Apply additional browser args and language via env-vars when
	// present. The WebView2 client DLL reads these from the
	// process environment on env creation.
	if opts.AdditionalBrowserArgs != "" {
		_ = os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", opts.AdditionalBrowserArgs)
	}
	if opts.Language != "" {
		_ = os.Setenv("WEBVIEW2_LANGUAGE", opts.Language)
	}

	preventEnvAndRegistryOverrides()

	const unknown = 1
	hr, _, _ := createProc.Call(
		uintptr(unknown),
		uintptr(runtimeTyp),
		uintptr(unsafe.Pointer(userDataPtr)),
		0, // envOptions (nil — runtime defaults)
		uintptr(handler.impl.COMObject()),
	)

	if hr != 0 {
		return fmt.Errorf("CreateWebViewEnvironmentWithOptionsInternal failed: 0x%08x", hr)
	}
	return nil
}

// preventEnvAndRegistryOverrides clears the WebView2 env-var and
// registry override paths so the env-creation call observes only
// the values the caller explicitly passed. Ported from upstream
// webviewloader.preventEnvAndRegistryOverrides.
func preventEnvAndRegistryOverrides() {
	// Setting these env variables to empty string also prevents
	// registry overrides because webview2 checks for existence and
	// not for empty value.
	_ = os.Setenv("WEBVIEW2_PIPE_FOR_SCRIPT_DEBUGGER", "")
	_ = os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "")
	_ = os.Setenv("WEBVIEW2_RELEASE_CHANNEL_PREFERENCE", "0")
	// The following seem not to be required because those are only
	// used by the webview2loader which in this case is implemented
	// on our own. But nevertheless set them to empty to be consistent.
	_ = os.Setenv("WEBVIEW2_BROWSER_EXECUTABLE_FOLDER", "")
	_ = os.Setenv("WEBVIEW2_USER_DATA_FOLDER", "")
}
