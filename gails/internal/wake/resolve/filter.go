package resolve

import (
	"github.com/gailsapp/gails/internal/wake/ast"
	"github.com/gailsapp/gails/internal/wake/platform"
)

func FilterPlatforms(tf *ast.Taskfile) {
	for _, task := range tf.Tasks {
		if !platform.Filter(task.Platforms) {
			task.Cmds = nil
			task.Deps = nil
		}
	}
}
