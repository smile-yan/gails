//go:build windows

package w32helper

import "testing"

func TestOle32CoInitializeEx_Exported(t *testing.T) {
	// The package must expose the symbol that pkg/w32/ole32.go currently
	// imports from upstream: a syscall.Proc named Ole32CoInitializeEx.
	// We don't invoke it (the test environment may not have COM initialized);
	// we only assert the proc is findable.
	if Ole32CoInitializeEx == nil {
		t.Fatal("Ole32CoInitializeEx proc is nil")
	}
}
