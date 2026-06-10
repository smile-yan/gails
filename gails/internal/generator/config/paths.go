package config

// GailsAppPkgPath is the official import path of Wails v3's application package.
const GailsAppPkgPath = "github.com/gailsapp/gails/v3/pkg/application"

// GailsInternalPkgPath is the official import path of Wails v3's internal package.
const GailsInternalPkgPath = "github.com/gailsapp/gails/v3/internal"

// SystemPaths holds resolved paths of required system packages.
type SystemPaths struct {
	ContextPackage     string
	ApplicationPackage string
	InternalPackage    string
}
