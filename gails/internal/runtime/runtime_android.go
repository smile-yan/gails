//go:build android

package runtime

// Android uses window.gails.invoke which is set up via addJavascriptInterface in WailsJSBridge
// We need to log the state to debug why it's not being detected
var invoke = `
console.log('[Wails Android Runtime] Injecting runtime, window.gails exists:', !!window.gails);
console.log('[Wails Android Runtime] window.gails.invoke exists:', !!(window.gails && window.gails.invoke));
window._gails.invoke=function(m){
    console.log('[Wails Android Runtime] _gails.invoke called:', m);
    return window.gails.invoke(typeof m==='string'?m:JSON.stringify(m));
};
console.log('[Wails Android Runtime] Runtime injection complete');
`
var flags = ""
