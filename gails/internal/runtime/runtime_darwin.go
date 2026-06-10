//go:build darwin

package runtime

var invoke = "window._gails.invoke=function(msg){window.webkit.messageHandlers.external.postMessage(msg);};"
