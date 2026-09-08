package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// BuildVersion is set via ldflags during build. When empty, Version returns
// the value embedded from the VERSION file.
var BuildVersion string

// Version returns the current ralph version.
func Version() string {
	if BuildVersion != "" {
		return BuildVersion
	}
	return strings.TrimSpace(versionFile)
}
