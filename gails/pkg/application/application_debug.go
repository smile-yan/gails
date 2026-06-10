//go:build !production

package application

import (
	"github.com/go-git/go-git/v5"
	"github.com/gailsapp/gails/internal/lo"
	"github.com/gailsapp/gails/internal/version"
	"path/filepath"
	"runtime/debug"
)

// BuildSettings contains the build settings for the application
var BuildSettings map[string]string

// BuildInfo contains the build info for the application
var BuildInfo *debug.BuildInfo

func init() {
	var ok bool
	BuildInfo, ok = debug.ReadBuildInfo()
	if !ok {
		return
	}
	BuildSettings = lo.Associate(BuildInfo.Settings, func(setting debug.BuildSetting) (string, string) {
		return setting.Key, setting.Value
	})
}

// We use this to patch the application to production mode.
func newApplication(options Options) *App {
	result := &App{
		isDebugMode: true,
		options:     options,
	}
	result.init()
	return result
}

func (a *App) logStartup() {
	var args []any

	// BuildInfo is nil when build with garble
	if BuildInfo == nil {
		return
	}

	gailsPackage, _ := lo.Find(BuildInfo.Deps, func(dep *debug.Module) bool {
		return dep.Path == "github.com/gailsapp/gails"
	})

	gailsVersion := version.String()
	if gailsPackage != nil && gailsPackage.Replace != nil {
		gailsVersion = "(local) => " + filepath.ToSlash(gailsPackage.Replace.Path)
		// Get the latest commit hash
		repo, err := git.PlainOpen(filepath.Join(gailsPackage.Replace.Path, ".."))
		if err == nil {
			head, err := repo.Head()
			if err == nil {
				gailsVersion += " (" + head.Hash().String()[:8] + ")"
			}
		}
	}
	args = append(args, "Wails", gailsVersion)
	args = append(args, "Compiler", BuildInfo.GoVersion)
	for key, value := range BuildSettings {
		args = append(args, key, value)
	}

	a.info("Build Info:", args...)
}
