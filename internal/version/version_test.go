package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
	assert.Equal(t, 1, patch, "Patch version should be 1")
}

func TestVersion_PatchBumpApplied(t *testing.T) {
	versionStr := Version()

	parts := strings.Split(versionStr, ".")
	require.Len(t, parts, 3, "Version should have 3 parts")

	var major, minor, patch int
	_, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "Version should be parseable")

	assert.Equal(t, "5.10.1", versionStr, "Version should be bumped to 5.10.1")
}

func TestVersion_MatchesChartAppVersion(t *testing.T) {
	// Read VERSION file
	versionData, err := os.ReadFile("VERSION")
	require.NoError(t, err, "Should read VERSION file")
	versionStr := strings.TrimSpace(string(versionData))

	// Parse version to ensure it's valid semver
	var major, minor, patch int
	_, err = fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "VERSION should be valid semver")

	// Read Chart.yaml
	chartPath := filepath.Join("..", "..", "charts", "ralph-webhook", "Chart.yaml")
	chartData, err := os.ReadFile(chartPath)
	require.NoError(t, err, "Should read Chart.yaml")

	var chart struct {
		Version    string `yaml:"version"`
		AppVersion string `yaml:"appVersion"`
	}
	err = yaml.Unmarshal(chartData, &chart)
	require.NoError(t, err, "Should parse Chart.yaml YAML")

	// Remove quotes from appVersion if present
	appVersion := strings.Trim(chart.AppVersion, "\"")
	assert.Equal(t, versionStr, appVersion, "Chart appVersion should match VERSION")

	// Validate chart version is also semver
	var chartMajor, chartMinor, chartPatch int
	_, err = fmt.Sscanf(chart.Version, "%d.%d.%d", &chartMajor, &chartMinor, &chartPatch)
	assert.NoError(t, err, "Chart version should be valid semver")
}
