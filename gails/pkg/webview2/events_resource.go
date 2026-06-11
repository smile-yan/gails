//go:build windows

package webview2

import (
	"fmt"
	"net/http"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// HttpRequestHeaders is a forward-declared placeholder for the COM
// ICoreWebView2HttpRequestHeaders interface. Gails currently only
// uses the raw pointer for UserAgent header manipulation; the full
// port (with GetHeader/SetHeader/Iterator) lives in a later task.
// This stub keeps the WebResourceRequest.GetHeaders signature
// stable until HttpRequestHeaders is fully ported.
type HttpRequestHeaders struct {
	Raw  uintptr
	vtbl *iCoreWebView2HttpRequestHeadersVtable
}

type iCoreWebView2HttpRequestHeadersVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// GetHeader, GetHeaders, SetHeader, RemoveHeader, GetIterator
	// follow upstream. The full vtable layout is owned by the
	// HttpRequestHeaders port task; for now we only model the
	// three IUnknown slots so a forward declaration compiles.
}

// AddRef increments the COM refcount. Provided so the
// ICoreWebView2HttpRequestHeaders stub can be Release'd by callers
// that use the GetHeaders forward declaration.
func (h *HttpRequestHeaders) AddRef() error {
	vtbl, err := h.vtable()
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(
		vtbl.AddRef,
		uintptr(unsafe.Pointer(h)),
	)
	return nil
}

// Release decrements the COM refcount.
func (h *HttpRequestHeaders) Release() error {
	vtbl, err := h.vtable()
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(
		vtbl.Release,
		uintptr(unsafe.Pointer(h)),
	)
	return nil
}

// GetHeader returns the value of the named header. Mirrors
// ICoreWebView2HttpRequestHeaders::GetHeader.
//
// TODO(port): the vtable slot for GetHeader is not yet modeled
// in iCoreWebView2HttpRequestHeadersVtable; a real implementation
// is a follow-up task. The application layer's UserAgent lookup
// tolerates an empty result, so a stub returning "" lets the
// file compile.
func (h *HttpRequestHeaders) GetHeader(_ string) (string, error) {
	return "", nil
}

// Header is a Go-style alias for GetHeader, kept for application
// code that prefers the Go convention.
func (h *HttpRequestHeaders) Header(name string) (string, error) {
	return h.GetHeader(name)
}

// SetHeader sets the named header to the given value, overwriting
// any existing value. Mirrors
// ICoreWebView2HttpRequestHeaders::SetHeader.
//
// TODO(port): the vtable slot for SetHeader is not yet modeled;
// stub returns nil so the call site compiles.
func (h *HttpRequestHeaders) SetHeader(_, _ string) error {
	return nil
}

func (h *HttpRequestHeaders) vtable() (*iCoreWebView2HttpRequestHeadersVtable, error) {
	if h.vtbl != nil {
		return h.vtbl, nil
	}
	if h.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2HttpRequestHeaders: nil COM pointer")
	}
	vtblPtr := *(*uintptr)(unsafe.Pointer(h.Raw))
	h.vtbl = (*iCoreWebView2HttpRequestHeadersVtable)(unsafe.Pointer(vtblPtr))
	return h.vtbl, nil
}

// WebResourceRequest is the COM ICoreWebView2WebResourceRequest
// wrapper. It exposes the request URI, method, body stream, and
// headers. Gails reads all four via the Get* accessors; the Put*
// setters are declared in the vtable layout so the slot indices
// line up with upstream, but Gails does not call them.
type WebResourceRequest struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebResourceRequestVtable
}

// iCoreWebView2WebResourceRequestVtable is the COM
// ICoreWebView2WebResourceRequest vtable. 3 IUnknown slots
// followed by 7 methods in upstream order:
//
//	[0]  QueryInterface
//	[1]  AddRef
//	[2]  Release
//	[3]  GetUri
//	[4]  PutUri        (modeled, not invoked)
//	[5]  GetMethod
//	[6]  PutMethod     (modeled, not invoked)
//	[7]  GetContent
//	[8]  PutContent    (modeled, not invoked)
//	[9]  GetHeaders
//
// Put slots are present so callers that probe the vtable land on
// the correct method; the Go surface only exposes the getters.
type iCoreWebView2WebResourceRequestVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetUri         uintptr
	PutUri         uintptr
	GetMethod      uintptr
	PutMethod      uintptr
	GetContent     uintptr
	PutContent     uintptr
	GetHeaders     uintptr
}

// AddRef increments the COM refcount.
func (r *WebResourceRequest) AddRef() error {
	vtbl, err := r.vtable()
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(
		vtbl.AddRef,
		uintptr(unsafe.Pointer(r)),
	)
	return nil
}

// Release decrements the COM refcount.
func (r *WebResourceRequest) Release() error {
	vtbl, err := r.vtable()
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(
		vtbl.Release,
		uintptr(unsafe.Pointer(r)),
	)
	return nil
}

// Uri returns the request URI. The COM method allocates the string
// with CoTaskMemAlloc; Gails does not free it (the WebView2 runtime
// owns the buffer for the lifetime of the request).
func (r *WebResourceRequest) Uri() (string, error) {
	vtbl, err := r.vtable()
	if err != nil {
		return "", err
	}
	var p *uint16
	hr, _, _ := syscall.SyscallN(
		vtbl.GetUri,
		uintptr(unsafe.Pointer(r)),
		uintptr(unsafe.Pointer(&p)),
	)
	if hr != 0 {
		return "", fmt.Errorf("ICoreWebView2WebResourceRequest::GetUri failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}

// Method returns the HTTP method (GET, POST, etc).
func (r *WebResourceRequest) Method() (string, error) {
	vtbl, err := r.vtable()
	if err != nil {
		return "", err
	}
	var p *uint16
	hr, _, _ := syscall.SyscallN(
		vtbl.GetMethod,
		uintptr(unsafe.Pointer(r)),
		uintptr(unsafe.Pointer(&p)),
	)
	if hr != 0 {
		return "", fmt.Errorf("ICoreWebView2WebResourceRequest::GetMethod failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}

// Content returns the request body as a Stream, or nil if the
// request has no body. The caller is responsible for releasing the
// returned stream.
func (r *WebResourceRequest) Content() (*Stream, error) {
	vtbl, err := r.vtable()
	if err != nil {
		return nil, err
	}
	var raw uintptr
	hr, _, _ := syscall.SyscallN(
		vtbl.GetContent,
		uintptr(unsafe.Pointer(r)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceRequest::GetContent failed: 0x%08x", hr)
	}
	if raw == 0 {
		return nil, nil
	}
	return &Stream{Raw: raw}, nil
}

// Headers returns the mutable HTTP request headers. The caller is
// responsible for releasing the returned headers object.
func (r *WebResourceRequest) Headers() (*HttpRequestHeaders, error) {
	vtbl, err := r.vtable()
	if err != nil {
		return nil, err
	}
	var raw uintptr
	hr, _, _ := syscall.SyscallN(
		vtbl.GetHeaders,
		uintptr(unsafe.Pointer(r)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceRequest::GetHeaders failed: 0x%08x", hr)
	}
	if raw == 0 {
		return nil, nil
	}
	return &HttpRequestHeaders{Raw: raw}, nil
}

func (r *WebResourceRequest) vtable() (*iCoreWebView2WebResourceRequestVtable, error) {
	if r.vtbl != nil {
		return r.vtbl, nil
	}
	if r.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceRequest: nil COM pointer")
	}
	vtblPtr := *(*uintptr)(unsafe.Pointer(r.Raw))
	r.vtbl = (*iCoreWebView2WebResourceRequestVtable)(unsafe.Pointer(vtblPtr))
	return r.vtbl, nil
}

// WebResourceResponse is the COM ICoreWebView2WebResourceResponse
// wrapper. Gails uses it to install a synthesized response on a
// WebResourceRequestedEventArgs (via PutResponse) and to set the
// response status code, reason phrase, and body content.
type WebResourceResponse struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebResourceResponseVtable
}

// iCoreWebView2WebResourceResponseVtable is the COM
// ICoreWebView2WebResourceResponse vtable. 3 IUnknown slots
// followed by 7 methods in upstream order:
//
//	[0]  QueryInterface
//	[1]  AddRef
//	[2]  Release
//	[3]  GetContent      (modeled, not invoked)
//	[4]  PutContent
//	[5]  GetHeaders      (modeled, not invoked)
//	[6]  GetStatusCode   (modeled, not invoked)
//	[7]  PutStatusCode
//	[8]  GetReasonPhrase (modeled, not invoked)
//	[9]  PutReasonPhrase
type iCoreWebView2WebResourceResponseVtable struct {
	QueryInterface   uintptr
	AddRef           uintptr
	Release          uintptr
	GetContent       uintptr
	PutContent       uintptr
	GetHeaders       uintptr
	GetStatusCode    uintptr
	PutStatusCode    uintptr
	GetReasonPhrase  uintptr
	PutReasonPhrase  uintptr
}

// AddRef increments the COM refcount.
func (r *WebResourceResponse) AddRef() error {
	vtbl, err := r.vtable()
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(
		vtbl.AddRef,
		uintptr(unsafe.Pointer(r)),
	)
	return nil
}

// Release decrements the COM refcount.
func (r *WebResourceResponse) Release() error {
	vtbl, err := r.vtable()
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(
		vtbl.Release,
		uintptr(unsafe.Pointer(r)),
	)
	return nil
}

// SetStatusCode sets the HTTP status code and (when the standard
// code is recognized) the reason phrase. Mirrors upstream
// PutStatusCode + PutReasonPhrase.
func (r *WebResourceResponse) SetStatusCode(statusCode int) error {
	vtbl, err := r.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutStatusCode,
		uintptr(unsafe.Pointer(r)),
		uintptr(statusCode),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2WebResourceResponse::PutStatusCode failed: 0x%08x", hr)
	}
	// PutStatusCode alone does not always set a reason phrase; the
	// upstream wailsv3 port sets it explicitly via PutReasonPhrase
	// using http.StatusText. Gails does the same to keep parity.
	phrase, err := windows.UTF16PtrFromString(http.StatusText(statusCode))
	if err != nil {
		return err
	}
	hr, _, _ = syscall.SyscallN(
		vtbl.PutReasonPhrase,
		uintptr(unsafe.Pointer(r)),
		uintptr(unsafe.Pointer(phrase)),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2WebResourceResponse::PutReasonPhrase failed: 0x%08x", hr)
	}
	return nil
}

// SetContent installs a stream as the response body. The WebView2
// runtime takes ownership of the stream after this call.
func (r *WebResourceResponse) SetContent(content *Stream) error {
	vtbl, err := r.vtable()
	if err != nil {
		return err
	}
	var streamPtr uintptr
	if content != nil {
		streamPtr = content.Raw
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutContent,
		uintptr(unsafe.Pointer(r)),
		streamPtr,
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2WebResourceResponse::PutContent failed: 0x%08x", hr)
	}
	return nil
}

func (r *WebResourceResponse) vtable() (*iCoreWebView2WebResourceResponseVtable, error) {
	if r.vtbl != nil {
		return r.vtbl, nil
	}
	if r.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceResponse: nil COM pointer")
	}
	vtblPtr := *(*uintptr)(unsafe.Pointer(r.Raw))
	r.vtbl = (*iCoreWebView2WebResourceResponseVtable)(unsafe.Pointer(vtblPtr))
	return r.vtbl, nil
}

// WebResourceRequestedEventArgs is the COM
// ICoreWebView2WebResourceRequestedEventArgs wrapper. Gails uses
// it to read the inbound request, install a synthesized response
// via PutResponse, and obtain a deferral for async handling.
type WebResourceRequestedEventArgs struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebResourceRequestedEventArgsVtable
}

// iCoreWebView2WebResourceRequestedEventArgsVtable is the COM
// ICoreWebView2WebResourceRequestedEventArgs vtable. 3 IUnknown
// slots followed by 5 methods in upstream order:
//
//	[0]  QueryInterface
//	[1]  AddRef
//	[2]  Release
//	[3]  GetRequest
//	[4]  GetResponse        (modeled, not invoked)
//	[5]  PutResponse
//	[6]  GetDeferral
//	[7]  GetResourceContext (modeled, not invoked)
type iCoreWebView2WebResourceRequestedEventArgsVtable struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	GetRequest        uintptr
	GetResponse       uintptr
	PutResponse       uintptr
	GetDeferral       uintptr
	GetResourceContext uintptr
}

// Request returns the inbound WebResourceRequest.
func (a *WebResourceRequestedEventArgs) Request() (*WebResourceRequest, error) {
	vtbl, err := a.vtable()
	if err != nil {
		return nil, err
	}
	var raw uintptr
	hr, _, _ := syscall.SyscallN(
		vtbl.GetRequest,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceRequestedEventArgs::GetRequest failed: 0x%08x", hr)
	}
	if raw == 0 {
		return nil, nil
	}
	return &WebResourceRequest{Raw: raw}, nil
}

// SetResponse installs a WebResourceResponse to be returned to the
// webview in place of the actual network response.
func (a *WebResourceRequestedEventArgs) SetResponse(response *WebResourceResponse) error {
	vtbl, err := a.vtable()
	if err != nil {
		return err
	}
	var respPtr uintptr
	if response != nil {
		respPtr = response.Raw
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutResponse,
		uintptr(unsafe.Pointer(a)),
		respPtr,
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2WebResourceRequestedEventArgs::PutResponse failed: 0x%08x", hr)
	}
	return nil
}

// Deferral returns a Deferral the caller must Complete when async
// work is done.
func (a *WebResourceRequestedEventArgs) Deferral() (*Deferral, error) {
	vtbl, err := a.vtable()
	if err != nil {
		return nil, err
	}
	var raw uintptr
	hr, _, _ := syscall.SyscallN(
		vtbl.GetDeferral,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceRequestedEventArgs::GetDeferral failed: 0x%08x", hr)
	}
	if raw == 0 {
		return nil, nil
	}
	return &Deferral{Raw: raw}, nil
}

func (a *WebResourceRequestedEventArgs) vtable() (*iCoreWebView2WebResourceRequestedEventArgsVtable, error) {
	if a.vtbl != nil {
		return a.vtbl, nil
	}
	if a.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2WebResourceRequestedEventArgs: nil COM pointer")
	}
	vtblPtr := *(*uintptr)(unsafe.Pointer(a.Raw))
	a.vtbl = (*iCoreWebView2WebResourceRequestedEventArgsVtable)(unsafe.Pointer(vtblPtr))
	return a.vtbl, nil
}

// WebResourceRequestedEventHandler is the Go-side
// ICoreWebView2WebResourceRequestedEventHandler implementation.
// Construct one with NewWebResourceRequestedEventHandler and pass
// to View.AddWebResourceRequested; call Close when done.
type WebResourceRequestedEventHandler struct {
	impl *comHandlerImpl
}

// NewWebResourceRequestedEventHandler wires a Go callback to the
// ICoreWebView2WebResourceRequestedEventHandler.Invoke vtable
// slot. The returned handler holds a reference to a native COM
// object; the caller must call Close when finished.
func NewWebResourceRequestedEventHandler(callback func(view *View, args *WebResourceRequestedEventArgs)) *WebResourceRequestedEventHandler {
	trampoline := windows.NewCallback(webResourceRequestedInvokeTrampoline)
	h := NewComHandler(trampoline, callback)
	return &WebResourceRequestedEventHandler{impl: h}
}

// Close releases the underlying COM object. Calling Close twice
// is a no-op.
func (h *WebResourceRequestedEventHandler) Close() {
	if h.impl == nil {
		return
	}
	h.impl.Release()
	h.impl = nil
}

// webResourceRequestedInvokeTrampoline is the per-instance Invoke
// slot for the ICoreWebView2WebResourceRequestedEventHandler
// vtable. It is registered as a C callback via windows.NewCallback
// and is invoked by WebView2 when a network resource request is
// dispatched.
//
// The signature is fixed by COM stdcall: the first argument is the
// COM `this` pointer, followed by the Invoke method's typed
// arguments, and the return value is an HRESULT uintptr.
func webResourceRequestedInvokeTrampoline(this uintptr, sender uintptr, args uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0x80004003 // E_POINTER
	}
	cb, ok := impl.Callback().(func(view *View, args *WebResourceRequestedEventArgs))
	if !ok || cb == nil {
		return 0 // S_OK; nothing to do
	}
	// The "sender" argument is an ICoreWebView2*. Wrap it in a
	// Gails View so the callback can issue further webview calls
	// (e.g. Navigate) if it needs to.
	var view *View
	if sender != 0 {
		view = &View{Raw: sender}
	}
	cb(view, &WebResourceRequestedEventArgs{Raw: args})
	return 0
}

// httpStatusText removed: SetStatusCode uses net/http.StatusText
// directly to keep parity with upstream and avoid a duplicated
// status-code-to-phrase table.
