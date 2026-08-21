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

func TestBumpMinor(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"increments minor", "9.0.2", "9.1.0"},
		{"increments minor from 0", "1.0.0", "1.1.0"},
		{"handles two digit minor", "9.99.0", "9.100.0"},
		{"resets patch to 0", "2.4.6", "2.5.0"},
		{"passes through non-numeric parts", "a.b.c", "a.b.c"},
		{"passes through non-numeric patch", "1.2.abc", "1.2.abc"},
		{"passes through prerelease suffix", "1.2.3-rc1", "1.2.3-rc1"},
		{"passes through trailing dot", "1.2.", "1.2."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BumpMinor(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
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
