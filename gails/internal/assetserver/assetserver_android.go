//go:build android

package assetserver

import "net/url"

// Android uses https://gails.localhost as the base URL
// This matches the WebViewAssetLoader domain configuration
var baseURL = url.URL{
	Scheme: "https",
	Host:   "gails.localhost",
}
