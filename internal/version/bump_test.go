package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBumpPatch(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"patch bump", "5.1.2", "5.1.3"},
		{"patch bump zero", "1.0.0", "1.0.1"},
		{"patch bump with leading zeros", "1.0.9", "1.0.10"},
		{"invalid version returns same", "invalid", "invalid"},
		{"invalid version partial", "1.2", "1.2"},
		{"four parts returns same", "1.2.3.4", "1.2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bumpPatch(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBumpPatchIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	versionFile := filepath.Join(tmpDir, "VERSION")
	chartFile := filepath.Join(tmpDir, "Chart.yaml")

	err := os.WriteFile(versionFile, []byte("5.1.2\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(chartFile, []byte(`apiVersion: v2
name: ralph-webhook
version: 1.0.12
appVersion: "5.1.2"
`), 0644)
	require.NoError(t, err)

	originalVersionFile := versionFilePath
	originalChartFile := chartYAMLFilePath
	versionFilePath = versionFile
	chartYAMLFilePath = chartFile
	defer func() {
		versionFilePath = originalVersionFile
		chartYAMLFilePath = originalChartFile
	}()

	err = BumpPatch()
	require.NoError(t, err)

	versionData, err := os.ReadFile(versionFile)
	require.NoError(t, err)
	assert.Equal(t, "5.1.3\n", string(versionData))

	chartData, err := os.ReadFile(chartFile)
	require.NoError(t, err)
	assert.Contains(t, string(chartData), `appVersion: "5.1.3"`)
	assert.Contains(t, string(chartData), "version: 1.0.13")
}
