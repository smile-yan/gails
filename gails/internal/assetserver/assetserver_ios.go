//go:build ios

package assetserver

import "net/url"

var baseURL = url.URL{
	Scheme: "gails",
	Host:   "localhost",
}
