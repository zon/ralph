package version

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

func TestVersion_ChartAppVersionMatches(t *testing.T) {
	raw, err := os.ReadFile("../../charts/ralph-webhook/Chart.yaml")
	require.NoError(t, err, "Chart.yaml should be readable")

	var chart struct {
		Version    string `yaml:"version"`
		AppVersion string `yaml:"appVersion"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &chart))

	appVersion := strings.Trim(chart.AppVersion, `"`)
	assert.Equal(t, Version(), appVersion, "chart appVersion should match the app version")
}

func TestVersion_ChartVersionIsValidSemver(t *testing.T) {
	raw, err := os.ReadFile("../../charts/ralph-webhook/Chart.yaml")
	require.NoError(t, err, "Chart.yaml should be readable")

	var chart struct {
		Version string `yaml:"version"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &chart))

	var major, minor, patch int
	_, err = fmt.Sscanf(chart.Version, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err, "chart version should be a valid semver in format X.Y.Z")
}
