package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigGithubCommand(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-key.pem")
	err := os.WriteFile(tmpFile, []byte("test private key"), 0644)
	require.NoError(t, err, "failed to create temp file")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "config github parses successfully",
			args:    []string{"config", "github", tmpFile},
			wantErr: false,
		},
		{
			name:    "config github with context",
			args:    []string{"config", "github", tmpFile, "--context", "my-cluster"},
			wantErr: false,
		},
		{
			name:    "config github with namespace",
			args:    []string{"config", "github", tmpFile, "--namespace", "my-namespace"},
			wantErr: false,
		},
		{
			name:    "config github with both context and namespace",
			args:    []string{"config", "github", tmpFile, "--context", "my-cluster", "--namespace", "my-namespace"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err, "failed to create parser")

			_, err = parser.Parse(tt.args)
			if tt.wantErr {
				assert.Error(t, err, "Parse should return error")
			} else {
				require.NoError(t, err, "Parse should not return error")
			}
		})
	}
}

func TestConfigGithubFlags(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-key.pem")
	err := os.WriteFile(tmpFile, []byte("test private key"), 0644)
	require.NoError(t, err, "failed to create temp file")

	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "default values",
			args:              []string{"config", "github", tmpFile},
			expectedContext:   "",
			expectedNamespace: "",
		},
		{
			name:              "with context",
			args:              []string{"config", "github", tmpFile, "--context", "test-context"},
			expectedContext:   "test-context",
			expectedNamespace: "",
		},
		{
			name:              "with namespace",
			args:              []string{"config", "github", tmpFile, "--namespace", "test-namespace"},
			expectedContext:   "",
			expectedNamespace: "test-namespace",
		},
		{
			name:              "with both",
			args:              []string{"config", "github", tmpFile, "--context", "test-context", "--namespace", "test-namespace"},
			expectedContext:   "test-context",
			expectedNamespace: "test-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err, "failed to create parser")

			_, err = parser.Parse(tt.args)
			require.NoError(t, err, "failed to parse args")

			assert.Equal(t, tt.expectedContext, cmd.Config.Github.Context, "Context should match")
			assert.Equal(t, tt.expectedNamespace, cmd.Config.Github.Namespace, "Namespace should match")
		})
	}
}
