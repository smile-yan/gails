//go:build production

// CI: this file is invisible to default `go test`. CI must run
// `go test -tags production ./internal/runtime/...` to exercise the
// production-tagged build (Debug:false environment) and this test.

package runtime

import (
	"runtime"
	"strings"
	"testing"
)

func TestEnvironment_Prod(t *testing.T) {
	if !strings.Contains(environment, `"Debug":false`) {
		t.Errorf("prod environment should have Debug:false; got %q", environment)
	}
	if !strings.Contains(environment, `"OS":"`+runtime.GOOS+`"`) {
		t.Errorf("prod environment should have host OS %q; got %q", runtime.GOOS, environment)
	}
	if !strings.Contains(environment, `"Arch":"`+runtime.GOARCH+`"`) {
		t.Errorf("prod environment should have host Arch %q; got %q", runtime.GOARCH, environment)
	}
}
