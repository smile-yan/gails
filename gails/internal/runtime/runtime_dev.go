//go:build !production

package runtime

import (
	"fmt"
	"runtime"
)

var environment = fmt.Sprintf(`window._gails.environment={"OS":"%s","Arch":"%s","Debug":true};`, runtime.GOOS, runtime.GOARCH)
