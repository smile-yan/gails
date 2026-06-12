package operatingsystem

import "strings"

// OS contains information about the operating system
type OS struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	Version  string `json:"Version"`
	Branding string `json:"Branding"`
}

func (o *OS) AsLogSlice() []any {
	return []any{
		"ID", o.ID,
		"Name", o.Name,
		"Version", o.Version,
		"Branding", o.Branding,
	}
}

// Info retrieves information about the current platform
func Info() (*OS, error) {
	return platformInfo()
}

// parseOsRelease is a pure function kept in the cross-platform file so any
// host (not just linux) can unit-test it. Currently only the linux
// platformInfo caller uses it, but the format is generic across distros.
func parseOsRelease(osRelease string) *OS {

	// Default value
	var result OS
	result.ID = "Unknown"
	result.Name = "Unknown"
	result.Version = "Unknown"

	// Split into lines
	lines := strings.Split(osRelease, "\n")
	// Iterate lines
	for _, line := range lines {
		// Split each line by the equals char
		splitLine := strings.SplitN(line, "=", 2)
		// Check we have
		if len(splitLine) != 2 {
			continue
		}
		switch splitLine[0] {
		case "ID":
			result.ID = strings.ToLower(strings.Trim(splitLine[1], `"`))
		case "NAME":
			result.Name = strings.Trim(splitLine[1], `"`)
		case "VERSION_ID":
			result.Version = strings.Trim(splitLine[1], `"`)
		case "VERSION":
			result.Branding = strings.Trim(splitLine[1], `"`)
		}
	}
	return &result
}
