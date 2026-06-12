package commands

import (
	"github.com/gailsapp/gails/internal/github"
	"github.com/gailsapp/gails/internal/term"
	"github.com/gailsapp/gails/internal/version"
)

type ReleaseNotesOptions struct {
	Version  string `name:"v" description:"The version to show release notes for"`
	NoColour bool   `name:"n" description:"Disable colour output"`
}

func ReleaseNotes(options *ReleaseNotesOptions) error {
	if options.NoColour {
		term.DisableColor()
	}

	term.Header("Release Notes")

	if version.IsDev() {
		term.Println("Release notes are not available for development builds")
		return nil
	}

	currentVersion := version.String()
	if options.Version != "" {
		currentVersion = options.Version
	}

	releaseNotes := getReleaseNotesFunc(currentVersion, options.NoColour)
	term.Println(releaseNotes)
	return nil
}

// getReleaseNotesFunc is a package-level indirection so tests can stub
// the GitHub release-notes fetch. Follows the hook-override pattern from
// internal/operatingsystem/os_linux.go (readOsReleaseFile) and
// internal/commands/task_wrapper.go (runTaskFunc).
var getReleaseNotesFunc = github.GetReleaseNotes
