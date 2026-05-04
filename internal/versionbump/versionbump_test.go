package versionbump

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinorBump(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"simple bump", "1.2.3", "1.3.0"},
		{"zero minor", "1.0.3", "1.1.0"},
		{"zero patch", "1.2.0", "1.3.0"},
		{"double digit", "1.10.3", "1.11.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MinorBump(tt.version)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPatchBump(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"simple bump", "1.2.3", "1.2.4"},
		{"zero minor", "1.0.0", "1.0.1"},
		{"double digit patch", "1.2.10", "1.2.11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PatchBump(tt.version)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMinorBump_InvalidVersion(t *testing.T) {
	_, err := MinorBump("invalid")
	assert.Error(t, err)

	_, err = MinorBump("1.2")
	assert.Error(t, err)

	_, err = MinorBump("1.2.3.4")
	assert.Error(t, err)

	_, err = MinorBump("a.b.c")
	assert.Error(t, err)
}

func TestPatchBump_InvalidVersion(t *testing.T) {
	_, err := PatchBump("invalid")
	assert.Error(t, err)

	_, err = PatchBump("1.2")
	assert.Error(t, err)

	_, err = PatchBump("1.2.3.4")
	assert.Error(t, err)

	_, err = PatchBump("a.b.c")
	assert.Error(t, err)
}

func TestAppBump(t *testing.T) {
	result, err := AppBump("9.0.1")
	require.NoError(t, err)
	assert.Equal(t, "9.1.0", result)
}

func TestChartBump(t *testing.T) {
	result, err := ChartBump("2.0.3")
	require.NoError(t, err)
	assert.Equal(t, "2.0.4", result)
}