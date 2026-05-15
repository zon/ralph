package version

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version returns the current ralph version read from the VERSION file.
func Version() string {
	return strings.TrimSpace(versionFile)
}

// BumpMinor returns a new version string with the minor component incremented
// and the patch component reset to 0. It expects input in "major.minor.patch" semver format.
func BumpMinor(v string) string {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) != 3 {
		return v
	}
	var major, minor, patch int
	fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch)
	return fmt.Sprintf("%d.%d.0", major, minor+1)
}
