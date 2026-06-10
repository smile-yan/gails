//go:build linux && !android

package assetserver

import "net/url"

var baseURL = url.URL{
	Scheme: "gails",
	Host:   "localhost",
}
