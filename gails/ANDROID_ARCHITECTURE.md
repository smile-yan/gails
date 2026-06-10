# Gails v3 Android Architecture

## Executive Summary

This document provides a comprehensive technical architecture for Android support in Gails v3. The implementation enables Go applications to run natively on Android with an Android WebView frontend, maintaining the Gails philosophy of using web technologies for UI while leveraging Go for business logic.

Unlike iOS which uses CGO with Objective-C, Android uses JNI (Java Native Interface) to bridge between Java/Kotlin and Go. The Go code is compiled as a shared library (`.so`) that is loaded by the Android application at runtime.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Components](#core-components)
3. [Layer Architecture](#layer-architecture)
4. [File Structure](#file-structure)
5. [Implementation Details](#implementation-details)
6. [Build System](#build-system)
7. [JNI Bridge Details](#jni-bridge-details)
8. [Asset Serving](#asset-serving)
9. [JavaScript Bridge](#javascript-bridge)
10. [Security Considerations](#security-considerations)
11. [Configuration Options](#configuration-options)
12. [Debugging](#debugging)
13. [API Reference](#api-reference)
14. [Troubleshooting](#troubleshooting)
15. [Future Enhancements](#future-enhancements)

## Architecture Overview

### Design Principles

1. **Battery Efficiency First**: All architectural decisions prioritize battery life
2. **No Network Ports**: Asset serving happens in-process via `WebViewAssetLoader`
3. **JNI Bridge Pattern**: Java Activity hosts WebView, Go provides business logic
4. **Gails v3 Compatibility**: Maintain API compatibility with existing Gails v3 applications
5. **Follow Fyne's gomobile pattern**: Use `-buildmode=c-shared` for native library

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Android Application                       │
├─────────────────────────────────────────────────────────────┤
│                    Java/Android Layer                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              MainActivity (Activity)                 │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │            Android WebView                     │  │   │
│  │  │  ┌─────────────────────────────────────────┐  │  │   │
│  │  │  │         Web Application (HTML/JS)        │  │  │   │
│  │  │  └─────────────────────────────────────────┘  │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  │                                                      │   │
│  │  WailsBridge        WailsPathHandler   WailsJSBridge│   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                  JNI Bridge Layer                            │
│            System.loadLibrary("gails")                       │
├─────────────────────────────────────────────────────────────┤
│                    Go Runtime (libwails.so)                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                 Gails Application                     │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │  │
│  │  │App Logic │  │Services  │  │Asset Server      │  │  │
│  │  └──────────┘  └──────────┘  └──────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Comparison with iOS Architecture

| Aspect | iOS | Android |
|--------|-----|---------|
| Native Language | Objective-C | Java |
| Bridge Technology | CGO (C headers) | JNI |
| Build Mode | `-buildmode=c-archive` (.a) | `-buildmode=c-shared` (.so) |
| Entry Point | `main.m` calls `WailsIOSMain()` | `MainActivity` loads `libwails.so` |
| WebView | WKWebView | Android WebView |
| URL Scheme | `gails://localhost` | `https://gails.localhost` |
| Asset Interception | `WKURLSchemeHandler` | `WebViewAssetLoader` + `PathHandler` |
| JS → Native | `WKScriptMessageHandler` | `@JavascriptInterface` |
| Native → JS | `evaluateJavaScript:` | `evaluateJavascript()` |
| App Lifecycle | `UIApplicationDelegate` | `Activity` lifecycle methods |

## Core Components

### 1. Java Components

#### MainActivity (`MainActivity.java`)

**Purpose**: Android Activity that hosts the WebView and manages app lifecycle.

**Location**: `build/android/app/src/main/java/com/gails/app/MainActivity.java`

**Key Responsibilities**:
- Initialize the native Go library via `WailsBridge`
- Configure and manage the Android WebView
- Set up asset loading via `WebViewAssetLoader`
- Handle Android lifecycle events (onCreate, onResume, onPause, onDestroy)
- Execute JavaScript in the WebView when requested by Go

**Key Methods**:
```java
onCreate(Bundle)           // Initialize bridge, setup WebView
setupWebView()             // Configure WebView settings and handlers
loadApplication()          // Load initial URL (https://gails.localhost/)
executeJavaScript(String)  // Run JS code (called from Go via JNI)
onResume() / onPause()     // Lifecycle events forwarded to Go
onDestroy()                // Cleanup resources
onBackPressed()            // Handle back navigation
```

#### WailsBridge (`WailsBridge.java`)

**Purpose**: Manages the JNI connection between Java and Go.

**Location**: `build/android/app/src/main/java/com/gails/app/WailsBridge.java`

**Key Responsibilities**:
- Load the native library (`System.loadLibrary("gails")`)
- Declare and call native methods
- Manage callbacks for async operations
- Forward lifecycle events to Go

**Native Method Declarations**:
```java
private static native void nativeInit(WailsBridge bridge);
private static native void nativeShutdown();
private static native void nativeOnResume();
private static native void nativeOnPause();
private static native byte[] nativeServeAsset(String path, String method, String headers);
private static native String nativeHandleMessage(String message);
private static native String nativeGetAssetMimeType(String path);
```

**Key Methods**:
```java
initialize()              // Call nativeInit, set up Go runtime
shutdown()                // Call nativeShutdown, cleanup
serveAsset(path, method, headers)  // Get asset data from Go
handleMessage(message)    // Send message to Go, get response
getAssetMimeType(path)    // Get MIME type for asset
executeJavaScript(js)     // Execute JS (callable from Go)
emitEvent(name, data)     // Emit event to frontend
```

#### WailsPathHandler (`WailsPathHandler.java`)

**Purpose**: Implements `WebViewAssetLoader.PathHandler` to serve assets from Go.

**Location**: `build/android/app/src/main/java/com/gails/app/WailsPathHandler.java`

**Key Responsibilities**:
- Intercept all requests to `https://gails.localhost/*`
- Forward requests to Go's asset server via `WailsBridge`
- Return `WebResourceResponse` with asset data

**Key Method**:
```java
@Nullable
public WebResourceResponse handle(@NonNull String path) {
    // Normalize path (/ -> /index.html)
    // Call bridge.serveAsset(path, "GET", "{}")
    // Get MIME type via bridge.getAssetMimeType(path)
    // Return WebResourceResponse with data
}
```

#### WailsJSBridge (`WailsJSBridge.java`)

**Purpose**: JavaScript interface exposed to the WebView for Go communication.

**Location**: `build/android/app/src/main/java/com/gails/app/WailsJSBridge.java`

**Key Responsibilities**:
- Expose methods to JavaScript via `@JavascriptInterface`
- Forward messages from JavaScript to Go
- Support both sync and async message patterns

**JavaScript Interface Methods**:
```java
@JavascriptInterface
public String invoke(String message)  // Sync call to Go

@JavascriptInterface
public void invokeAsync(String callbackId, String message)  // Async call

@JavascriptInterface
public void log(String level, String message)  // Log to Android logcat

@JavascriptInterface
public String platform()  // Returns "android"

@JavascriptInterface
public boolean isDebug()  // Returns BuildConfig.DEBUG
```

**Usage from JavaScript**:
```javascript
// Synchronous call
const result = gails.invoke(JSON.stringify({type: 'call', ...}));

// Asynchronous call
gails.invokeAsync('callback-123', JSON.stringify({type: 'call', ...}));

// Logging
gails.log('info', 'Hello from JavaScript');

// Platform detection
if (gails.platform() === 'android') { ... }
```

### 2. Go Components

#### Application Layer (`application_android.go`)

**Purpose**: Main Go implementation for Android platform.

**Location**: `v3/pkg/application/application_android.go`

**Build Tag**: `//go:build android`

**Key Responsibilities**:
- Export JNI functions for Java to call
- Manage global application state
- Handle lifecycle events from Android
- Serve assets and process messages

**JNI Exports**:
```go
//export Java_com_wails_app_WailsBridge_nativeInit
func Java_com_wails_app_WailsBridge_nativeInit(env *C.JNIEnv, obj C.jobject, bridge C.jobject)

//export Java_com_wails_app_WailsBridge_nativeShutdown
func Java_com_wails_app_WailsBridge_nativeShutdown(env *C.JNIEnv, obj C.jobject)

//export Java_com_wails_app_WailsBridge_nativeOnResume
func Java_com_wails_app_WailsBridge_nativeOnResume(env *C.JNIEnv, obj C.jobject)

//export Java_com_wails_app_WailsBridge_nativeOnPause
func Java_com_wails_app_WailsBridge_nativeOnPause(env *C.JNIEnv, obj C.jobject)

//export Java_com_wails_app_WailsBridge_nativeServeAsset
func Java_com_wails_app_WailsBridge_nativeServeAsset(env *C.JNIEnv, obj C.jobject, path, method, headers *C.char) *C.char

//export Java_com_wails_app_WailsBridge_nativeHandleMessage
func Java_com_wails_app_WailsBridge_nativeHandleMessage(env *C.JNIEnv, obj C.jobject, message *C.char) *C.char

//export Java_com_wails_app_WailsBridge_nativeGetAssetMimeType
func Java_com_wails_app_WailsBridge_nativeGetAssetMimeType(env *C.JNIEnv, obj C.jobject, path *C.char) *C.char
```

**Platform Functions**:
```go
func (a *App) platformRun()      // Block forever, Android manages lifecycle
func (a *App) platformQuit()     // Signal quit
func (a *App) isDarkMode() bool  // Query Android dark mode
```

#### WebView Window (`webview_window_android.go`)

**Purpose**: Implements `webviewWindowImpl` interface for Android.

**Location**: `v3/pkg/application/webview_window_android.go`

**Build Tag**: `//go:build android`

**Key Methods**: Most methods are no-ops or return defaults since Android has a single fullscreen window.

```go
func (w *androidWebviewWindow) execJS(js string)     // Execute JavaScript
func (w *androidWebviewWindow) isFullscreen() bool   // Always true
func (w *androidWebviewWindow) size() (int, int)     // Device dimensions
func (w *androidWebviewWindow) setBackgroundColour(col RGBA)  // Set WebView bg
```

#### Asset Server (`assetserver_android.go`)

**Purpose**: Configure base URL for Android asset serving.

**Location**: `v3/internal/assetserver/assetserver_android.go`

**Build Tag**: `//go:build android`

```go
var baseURL = url.URL{
    Scheme: "https",
    Host:   "gails.localhost",
}
```

#### Other Platform Files

All these files have the `//go:build android` tag:

| File | Purpose |
|------|---------|
| `init_android.go` | Initialization (no `runtime.LockOSThread`) |
| `clipboard_android.go` | Clipboard operations (stub) |
| `dialogs_android.go` | File/message dialogs (stub) |
| `menu_android.go` | Menu handling (no-op) |
| `menuitem_android.go` | Menu items (no-op) |
| `screen_android.go` | Screen information |
| `mainthread_android.go` | Main thread dispatch |
| `signal_handler_android.go` | Signal handling (no-op) |
| `single_instance_android.go` | Single instance (via manifest) |
| `systemtray_android.go` | System tray (no-op) |
| `keys_android.go` | Keyboard handling (stub) |
| `events_common_android.go` | Event mapping |
| `messageprocessor_android.go` | Android-specific runtime methods |

## File Structure

```
v3/
├── ANDROID_ARCHITECTURE.md          # This document
├── pkg/
│   ├── application/
│   │   ├── application_android.go   # Main Android implementation
│   │   ├── application_options.go   # Contains AndroidOptions struct
│   │   ├── webview_window_android.go
│   │   ├── clipboard_android.go
│   │   ├── dialogs_android.go
│   │   ├── events_common_android.go
│   │   ├── init_android.go
│   │   ├── keys_android.go
│   │   ├── mainthread_android.go
│   │   ├── menu_android.go
│   │   ├── menuitem_android.go
│   │   ├── messageprocessor_android.go
│   │   ├── messageprocessor_mobile_stub.go  # Stub for non-mobile
│   │   ├── screen_android.go
│   │   ├── signal_handler_android.go
│   │   ├── signal_handler_types_android.go
│   │   ├── single_instance_android.go
│   │   └── systemtray_android.go
│   └── events/
│       └── events_android.go
├── internal/
│   └── assetserver/
│       ├── assetserver_android.go
│       └── webview/
│           └── request_android.go
└── examples/
    └── android/
        ├── main.go                  # Application entry point
        ├── greetservice.go          # Example service
        ├── go.mod
        ├── go.sum
        ├── Taskfile.yml             # Build orchestration
        ├── .gitignore
        ├── frontend/                # Web frontend (same as other platforms)
        │   ├── index.html
        │   ├── main.js
        │   ├── package.json
        │   └── ...
        └── build/
            ├── config.yml           # Build configuration
            ├── Taskfile.yml         # Common build tasks
            ├── android/
            │   ├── Taskfile.yml     # Android-specific tasks
            │   ├── build.gradle     # Root Gradle build
            │   ├── settings.gradle
            │   ├── gradle.properties
            │   ├── gradlew          # Gradle wrapper script
            │   ├── gradle/
            │   │   └── wrapper/
            │   │       └── gradle-wrapper.properties
            │   ├── scripts/
            │   │   └── deps/
            │   │       └── install_deps.go  # Dependency checker
            │   └── app/
            │       ├── build.gradle         # App Gradle build
            │       ├── proguard-rules.pro
            │       └── src/
            │           └── main/
            │               ├── AndroidManifest.xml
            │               ├── java/
            │               │   └── com/
            │               │       └── gails/
            │               │           └── app/
            │               │               ├── MainActivity.java
            │               │               ├── WailsBridge.java
            │               │               ├── WailsPathHandler.java
            │               │               └── WailsJSBridge.java
            │               ├── res/
            │               │   ├── layout/
            │               │   │   └── activity_main.xml
            │               │   ├── values/
            │               │   │   ├── strings.xml
            │               │   │   ├── colors.xml
            │               │   │   └── themes.xml
            │               │   └── mipmap-*/  # App icons
            │               ├── assets/        # Frontend assets (copied)
            │               └── jniLibs/
            │                   ├── arm64-v8a/
            │                   │   └── libwails.so  # Generated
            │                   └── x86_64/
            │                       └── libwails.so  # Generated
            ├── darwin/              # macOS build files
            ├── linux/               # Linux build files
            └── windows/             # Windows build files
```

## Implementation Details

### Application Startup Flow

```
1. Android OS launches MainActivity
   │
2. MainActivity.onCreate()
   │
   ├─> WailsBridge.initialize()
   │   │
   │   └─> System.loadLibrary("gails")
   │       │
   │       └─> Go runtime starts
   │           │
   │           └─> nativeInit() called
   │               │
   │               └─> globalApp = app (store reference)
   │
   ├─> setupWebView()
   │   │
   │   ├─> Configure WebSettings
   │   ├─> Create WebViewAssetLoader with WailsPathHandler
   │   ├─> Set WebViewClient for request interception
   │   └─> Add WailsJSBridge via addJavascriptInterface
   │
   └─> loadApplication()
       │
       └─> webView.loadUrl("https://gails.localhost/")
           │
           └─> WailsPathHandler.handle("/")
               │
               └─> WailsBridge.serveAsset("/index.html", ...)
                   │
                   └─> nativeServeAsset() (JNI to Go)
                       │
                       └─> Go AssetServer returns HTML
```

### Asset Request Flow

```
WebView requests: https://gails.localhost/main.js
        │
        ▼
WebViewClient.shouldInterceptRequest()
        │
        ▼
WebViewAssetLoader.shouldInterceptRequest()
        │
        ▼
WailsPathHandler.handle("/main.js")
        │
        ▼
WailsBridge.serveAsset("/main.js", "GET", "{}")
        │
        ▼
JNI call: nativeServeAsset(path, method, headers)
        │
        ▼
Go: serveAssetForAndroid(app, "/main.js")
        │
        ▼
Go: AssetServer reads from embed.FS
        │
        ▼
Return: byte[] data
        │
        ▼
WailsPathHandler creates WebResourceResponse
        │
        ▼
WebView renders content
```

### JavaScript to Go Message Flow

```
JavaScript: gails.invoke('{"type":"call","method":"Greet","args":["World"]}')
        │
        ▼
WailsJSBridge.invoke(message)  [@JavascriptInterface]
        │
        ▼
WailsBridge.handleMessage(message)
        │
        ▼
JNI call: nativeHandleMessage(message)
        │
        ▼
Go: handleMessageForAndroid(app, message)
        │
        ▼
Go: Parse JSON, route to service method
        │
        ▼
Go: Execute GreetService.Greet("World")
        │
        ▼
Return: '{"result":"Hello, World!"}'
        │
        ▼
JavaScript receives result
```

### Go to JavaScript Event Flow

```
Go: app.Event.Emit("time", "Mon, 01 Jan 2024 12:00:00")
        │
        ▼
Go: Call Java executeJavaScript via JNI callback
        │
        ▼
WailsBridge.emitEvent("time", "\"Mon, 01 Jan 2024 12:00:00\"")
        │
        ▼
JavaScript: window.gails._emit('time', "Mon, 01 Jan 2024 12:00:00")
        │
        ▼
Frontend event listeners notified
```

## Build System

### Prerequisites

1. **Go 1.21+** with CGO support
2. **Android SDK** with:
   - Platform Tools (adb)
   - Build Tools
   - Android Emulator
3. **Android NDK r19c+** (r26d recommended)
4. **Java JDK 11+**

### Environment Variables

```bash
# Required
export ANDROID_HOME=$HOME/Library/Android/sdk    # macOS
export ANDROID_HOME=$HOME/Android/Sdk            # Linux

# Optional (auto-detected if not set)
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/26.1.10909125

# Path additions
export PATH=$PATH:$ANDROID_HOME/platform-tools
export PATH=$PATH:$ANDROID_HOME/emulator
```

### Taskfile Commands

```bash
# Check/install dependencies
task android:install:deps

# Build Go shared library (default: arm64 for device)
task android:build

# Build for emulator (x86_64)
task android:build ARCH=x86_64

# Build for all architectures (fat APK)
task android:compile:go:all-archs

# Package into APK
task android:package

# Run on emulator
task android:run

# View logs
task android:logs

# Clean build artifacts
task android:clean
```

### Build Process Details

#### 1. Go Compilation

```bash
# Environment for arm64 (device)
export GOOS=android
export GOARCH=arm64
export CGO_ENABLED=1
export CC=$NDK/toolchains/llvm/prebuilt/$HOST/bin/aarch64-linux-android21-clang

# Build command
go build -buildmode=c-shared \
    -tags android \
    -o build/android/app/src/main/jniLibs/arm64-v8a/libwails.so
```

#### 2. Gradle Build

```bash
cd build/android
./gradlew assembleDebug
# Output: app/build/outputs/apk/debug/app-debug.apk
```

#### 3. Installation

```bash
adb install app-debug.apk
adb shell am start -n com.gails.app/.MainActivity
```

### Architecture Support

| Architecture | GOARCH | JNI Directory | Use Case |
|--------------|--------|---------------|----------|
| arm64-v8a | arm64 | `jniLibs/arm64-v8a/` | Physical devices (most common) |
| x86_64 | amd64 | `jniLibs/x86_64/` | Emulator |
| armeabi-v7a | arm | `jniLibs/armeabi-v7a/` | Older devices (optional) |
| x86 | 386 | `jniLibs/x86/` | Older emulators (optional) |

### Minimum SDK Configuration

```gradle
// build/android/app/build.gradle
android {
    defaultConfig {
        minSdk 21        // Android 5.0 (Lollipop) - 99%+ coverage
        targetSdk 34     // Android 14 - Required for Play Store
    }
}
```

## JNI Bridge Details

### JNI Function Naming Convention

JNI functions must follow this naming pattern:
```
Java_<package>_<class>_<method>
```

Example:
```go
//export Java_com_wails_app_WailsBridge_nativeInit
func Java_com_wails_app_WailsBridge_nativeInit(env *C.JNIEnv, obj C.jobject, bridge C.jobject)
```

Corresponds to Java:
```java
package com.gails.app;
class WailsBridge {
    private static native void nativeInit(WailsBridge bridge);
}
```

### JNI Type Mappings

| Java Type | JNI Type | Go CGO Type |
|-----------|----------|-------------|
| void | void | - |
| boolean | jboolean | C.jboolean |
| int | jint | C.jint |
| long | jlong | C.jlong |
| String | jstring | *C.char (via conversion) |
| byte[] | jbyteArray | *C.char (via conversion) |
| Object | jobject | C.jobject |

### String Conversion

```go
// Java String → Go string
goString := C.GoString((*C.char)(unsafe.Pointer(javaString)))

// Go string → Java String (return)
return C.CString(goString)  // Must be freed by Java
```

### Thread Safety

- JNI calls must be made from the thread that owns the JNI environment
- Go goroutines cannot directly call JNI methods
- Use channels or callbacks to communicate between goroutines and JNI thread

## Asset Serving

### WebViewAssetLoader Configuration

```java
assetLoader = new WebViewAssetLoader.Builder()
    .setDomain("gails.localhost")           // Custom domain
    .addPathHandler("/", new WailsPathHandler(bridge))  // All paths
    .build();
```

### URL Scheme

- **Base URL**: `https://gails.localhost/`
- **Why HTTPS**: Android's `WebViewAssetLoader` requires HTTPS for security
- **Domain**: `gails.localhost` is arbitrary but consistent with Gails conventions

### Path Normalization

```java
// In WailsPathHandler.handle()
if (path.isEmpty() || path.equals("/")) {
    path = "/index.html";
}
```

### MIME Type Detection

MIME types are determined by Go based on file extension. Fallback mapping in Java:

```java
private String getMimeType(String path) {
    if (path.endsWith(".html")) return "text/html";
    if (path.endsWith(".js")) return "application/javascript";
    if (path.endsWith(".css")) return "text/css";
    // ... etc
    return "application/octet-stream";
}
```

## JavaScript Bridge

### Exposed Interface

The `WailsJSBridge` is added to the WebView as:
```java
webView.addJavascriptInterface(new WailsJSBridge(bridge, webView), "gails");
```

This makes `window.gails` available in JavaScript.

### Security Considerations

1. **@JavascriptInterface annotation** is required for all exposed methods (Android 4.2+)
2. Only specific methods are exposed, not the entire object
3. Input validation should be performed on all received data

### Async Pattern

For non-blocking calls:

```javascript
// JavaScript side
const callbackId = 'cb_' + Date.now();
window.gails._callbacks[callbackId] = (result, error) => {
    if (error) reject(error);
    else resolve(result);
};
gails.invokeAsync(callbackId, message);

// Java side sends response via:
webView.evaluateJavascript(
    "window.gails._callback('" + callbackId + "', " + result + ", null);",
    null
);
```

## Security Considerations

### WebView Security

```java
WebSettings settings = webView.getSettings();
settings.setAllowFileAccess(false);          // No file:// access
settings.setAllowContentAccess(false);       // No content:// access
settings.setMixedContentMode(MIXED_CONTENT_NEVER_ALLOW);  // HTTPS only
```

### JNI Security

1. **No arbitrary code execution**: JNI methods have fixed signatures
2. **Input validation**: All strings from Java are validated in Go
3. **Memory safety**: Go's memory management prevents buffer overflows

### Asset Security

1. **Same-origin policy**: Assets only served from `gails.localhost`
2. **No external network**: All assets embedded, no remote fetching
3. **Content Security Policy**: Can be set via HTML headers

## Configuration Options

### AndroidOptions Struct

```go
type AndroidOptions struct {
    // DisableScroll disables scrolling in the WebView
    DisableScroll bool

    // DisableOverscroll disables the overscroll bounce effect
    DisableOverscroll bool

    // EnableZoom allows pinch-to-zoom in the WebView (default: false)
    EnableZoom bool

    // UserAgent sets a custom user agent string
    UserAgent string

    // BackgroundColour sets the background colour of the WebView
    BackgroundColour RGBA

    // DisableHardwareAcceleration disables hardware acceleration
    DisableHardwareAcceleration bool
}
```

### Usage

```go
app := application.New(application.Options{
    Name: "My App",
    Android: application.AndroidOptions{
        DisableOverscroll: true,
        BackgroundColour: application.NewRGB(27, 38, 54),
    },
})
```

### AndroidManifest.xml Configuration

```xml
<manifest>
    <uses-permission android:name="android.permission.INTERNET" />

    <application
        android:usesCleartextTraffic="true"  <!-- For localhost -->
        android:hardwareAccelerated="true">

        <activity
            android:name=".MainActivity"
            android:configChanges="orientation|screenSize|keyboardHidden"
            android:windowSoftInputMode="adjustResize">
        </activity>
    </application>
</manifest>
```

## Debugging

### Logcat Filtering

```bash
# All Gails logs
adb logcat -v time | grep -E "(Gails|WailsBridge|WailsActivity)"

# Using task
task android:logs
```

### WebView Debugging

Enable in debug builds:
```java
if (BuildConfig.DEBUG) {
    WebView.setWebContentsDebuggingEnabled(true);
}
```

Then in Chrome: `chrome://inspect/#devices`

### Go Debugging

```go
func androidLogf(level string, format string, a ...interface{}) {
    msg := fmt.Sprintf(format, a...)
    println(fmt.Sprintf("[Android/%s] %s", level, msg))
}
```

### Common Issues

1. **"UnsatisfiedLinkError"**: Library not found or wrong architecture
2. **"No implementation found"**: JNI function name mismatch
3. **Blank WebView**: Asset serving not working, check logcat

## API Reference

### Go API (Same as Desktop)

```go
// Create application
app := application.New(application.Options{
    Name: "App Name",
    Assets: application.AssetOptions{
        Handler: application.AssetFileServerFS(assets),
    },
    Services: []application.Service{
        application.NewService(&MyService{}),
    },
    Android: application.AndroidOptions{...},
})

// Run (blocks on Android)
app.Run()

// Emit events
app.Event.Emit("eventName", data)
```

### JavaScript API

```javascript
// Call Go service method
const result = await window.gails.Call.ByName('MyService.Greet', 'World');

// Platform detection
if (window.gails.System.Platform() === 'android') { ... }

// Events
window.gails.Events.On('eventName', (data) => { ... });
```

### Android-Specific Runtime Methods

```javascript
// Vibrate (haptic feedback)
window.gails.Call.ByName('Android.Haptics.Vibrate', {duration: 100});

// Show toast
window.gails.Call.ByName('Android.Toast.Show', {message: 'Hello!'});

// Get device info
const info = await window.gails.Call.ByName('Android.Device.Info');
```

## Troubleshooting

### Build Errors

**"NDK not found"**
```bash
# Set NDK path explicitly
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/26.1.10909125
```

**"undefined reference to JNI function"**
- Check function name matches exactly (case-sensitive)
- Ensure `//export` comment is directly above function

**"cannot find package"**
```bash
cd examples/android && go mod tidy
```

### Runtime Errors

**App crashes on startup**
1. Check logcat for stack trace
2. Verify library is in correct jniLibs directory
3. Check architecture matches device/emulator

**WebView shows blank**
1. Enable WebView debugging
2. Check Chrome DevTools for errors
3. Verify `https://gails.localhost/` resolves

**JavaScript bridge not working**
1. Check `gails` object exists: `console.log(window.gails)`
2. Verify `@JavascriptInterface` annotations present
3. Check for JavaScript errors in console

## Future Enhancements

### Phase 1: Core Stability
- [ ] Complete JNI callback implementation for Go → Java
- [ ] Full asset server integration
- [ ] Error handling and recovery
- [ ] Unit and integration tests

### Phase 2: Feature Parity
- [ ] Clipboard support
- [ ] File dialogs (via Storage Access Framework)
- [ ] Notifications
- [ ] Deep linking

### Phase 3: Android-Specific Features
- [ ] Material Design 3 theming integration
- [ ] Edge-to-edge display support
- [ ] Predictive back gesture
- [ ] Picture-in-Picture mode
- [ ] Widgets

### Phase 4: Advanced Features
- [ ] Background services
- [ ] Push notifications (FCM)
- [ ] Biometric authentication
- [ ] App Shortcuts
- [ ] Wear OS companion

## Conclusion

This architecture provides a solid foundation for Android support in Gails v3. The design prioritizes:

1. **Compatibility**: Same Go code runs on all platforms
2. **Performance**: No network overhead, native rendering
3. **Security**: Sandboxed WebView, validated inputs
4. **Maintainability**: Clear separation of concerns

The implementation follows Android best practices while maintaining the simplicity that Gails developers expect. The JNI bridge pattern, while more complex than iOS's CGO approach, provides robust interoperability between Java and Go.

### Key Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| Java Activity | ✅ Complete | MainActivity with WebView |
| JNI Bridge | ✅ Complete | WailsBridge with native methods |
| Asset Handler | ✅ Complete | WailsPathHandler |
| JS Bridge | ✅ Complete | WailsJSBridge |
| Go Platform Files | ✅ Complete | All *_android.go files |
| Taskfile | ✅ Complete | Build orchestration |
| Gradle Project | ✅ Complete | App structure |
| JNI Implementation | 🔄 Partial | Needs Go → Java callbacks |
| Asset Server Integration | 🔄 Partial | Needs full wiring |
| Testing | ❌ Pending | Needs emulator testing |

---

*Document Version: 1.0*
*Last Updated: November 2024*
*Gails Version: v3-alpha*
