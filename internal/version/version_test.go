package version

import (
	_ "embed"
	"fmt"
	"os"
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

func TestBumpMinor_IncrementsMinorVersion(t *testing.T) {
	orig, err := os.ReadFile("/workspace/repo/internal/version/VERSION")
	require.NoError(t, err)

	bumped, err := BumpMinorString(strings.TrimSpace(string(orig)))
	require.NoError(t, err)

	var major, minor, patch int
	_, err = fmt.Sscanf(bumped, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err)

	var omajor, ominor, opatch int
	fmt.Sscanf(strings.TrimSpace(string(orig)), "%d.%d.%d", &omajor, &ominor, &opatch)

	assert.Equal(t, omajor, major, "major should be unchanged")
	assert.Equal(t, ominor+1, minor, "minor should be incremented")
	assert.Equal(t, opatch, patch, "patch should be unchanged")
}

func TestBumpMinor_InvalidVersionErrors(t *testing.T) {
	_, err := BumpMinorString("not-semver")
	assert.Error(t, err)
}
