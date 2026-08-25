package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version returns the current ralph version read from the VERSION file.
func Version() string {
	return strings.TrimSpace(versionFile)
}
