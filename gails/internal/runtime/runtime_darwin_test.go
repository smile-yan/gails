//go:build darwin

package runtime

import "testing"

// TestInvoke_Darwin locks in the byte-exact darwin invoke literal. Any
// change to the message-passing bridge (e.g. switching from webkit to
// chrome) must update this test intentionally.
func TestInvoke_Darwin(t *testing.T) {
	want := "window._gails.invoke=function(msg){window.webkit.messageHandlers.external.postMessage(msg);};"
	if invoke != want {
		t.Errorf("darwin invoke mismatch:\n  got  %q\n  want %q", invoke, want)
	}
}
