//go:build !production

package runtime

import (
	"runtime"
	"strings"
	"testing"
)

func TestEnvironment_Dev(t *testing.T) {
	if !strings.Contains(environment, `"Debug":true`) {
		t.Errorf("dev environment should have Debug:true; got %q", environment)
	}
	if !strings.Contains(environment, `"OS":"`+runtime.GOOS+`"`) {
		t.Errorf("dev environment should have host OS %q; got %q", runtime.GOOS, environment)
	}
	if !strings.Contains(environment, `"Arch":"`+runtime.GOARCH+`"`) {
		t.Errorf("dev environment should have host Arch %q; got %q", runtime.GOARCH, environment)
	}
}
