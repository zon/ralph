package version

import (
	_ "embed"
	"fmt"
	"strconv"
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
// Malformed input is returned unchanged.
func BumpMinor(v string) string {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) != 3 {
		return v
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	_, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return v
	}
	return fmt.Sprintf("%d.%d.0", major, minor+1)
}
