package version

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion_ReturnsValidSemver(t *testing.T) {
	versionStr := Version()

	require.NotEmpty(t, versionStr, "Version should not be empty")

	parts := strings.Split(versionStr, ".")
	require.Len(t, parts, 3, "Version should have 3 parts (major.minor.patch)")

	var major, minor, patch int
	_, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "Version should be a valid semver in format X.Y.Z")

	assert.Equal(t, 5, major, "Major version should be 5")
	assert.Equal(t, 10, minor, "Minor version should be 10")
	assert.Equal(t, 0, patch, "Patch version should be 0")
}

func TestVersion_PatchBumpApplied(t *testing.T) {
	versionStr := Version()

	parts := strings.Split(versionStr, ".")
	require.Len(t, parts, 3, "Version should have 3 parts")

	var major, minor, patch int
	_, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "Version should be parseable")

	assert.Equal(t, "5.10.0", versionStr, "Version should be bumped to 5.10.0")
}
