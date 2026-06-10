package runtime

import (
	"fmt"

	"encoding/json"
)

var runtimeInit = `window._gails=window._gails||{};window._gails.flags=window._gails.flags||{};window.gails=window.gails||{};`

func Core(flags map[string]any) string {
	flagsStr := ""
	if len(flags) > 0 {
		f, err := json.Marshal(flags)
		if err == nil {
			flagsStr += fmt.Sprintf("window._gails.flags=%s;", f)
		}
	}

	return runtimeInit + flagsStr + invoke + environment
}
