package version

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed VERSION
var versionSource string

func TestVersion_ReturnsValidSemver(t *testing.T) {
	versionStr := Version()

	require.NotEmpty(t, versionStr, "Version should not be empty")

	parts := strings.Split(versionStr, ".")
	require.Len(t, parts, 3, "Version should have 3 parts (major.minor.patch)")

	var major, minor, patch int
	_, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "Version should be a valid semver in format X.Y.Z")
}

func TestVersion_MatchesSourceFile(t *testing.T) {
	versionStr := Version()
	expectedVersion := strings.TrimSpace(versionSource)

	assert.Equal(t, expectedVersion, versionStr, "Version should match the VERSION file")
}

func TestVersion_PatchBumpApplied(t *testing.T) {
	versionStr := Version()
	expectedVersion := strings.TrimSpace(versionSource)

	parts := strings.Split(versionStr, ".")
	require.Len(t, parts, 3, "Version should have 3 parts")

	var major, minor, patch int
	_, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "Version should be parseable")

	assert.Equal(t, expectedVersion, versionStr, "Version should match the VERSION file")
}
