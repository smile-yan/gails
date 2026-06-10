package commands

import (
	"github.com/gailsapp/gails/v3/internal/version"
)

type VersionOptions struct{}

func Version(_ *VersionOptions) error {
	DisableFooter = true
	println(version.String())
	return nil
}
