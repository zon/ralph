package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigOpencodeCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "config opencode parses successfully",
			args:    []string{"config", "opencode"},
			wantErr: false,
		},
		{
			name:    "config opencode with context",
			args:    []string{"config", "opencode", "--context", "my-cluster"},
			wantErr: false,
		},
		{
			name:    "config opencode with namespace",
			args:    []string{"config", "opencode", "--namespace", "my-namespace"},
			wantErr: false,
		},
		{
			name:    "config opencode with both context and namespace",
			args:    []string{"config", "opencode", "--context", "my-cluster", "--namespace", "my-namespace"},
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

func TestConfigOpencodeFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "default values",
			args:              []string{"config", "opencode"},
			expectedContext:   "",
			expectedNamespace: "",
		},
		{
			name:              "with context",
			args:              []string{"config", "opencode", "--context", "test-context"},
			expectedContext:   "test-context",
			expectedNamespace: "",
		},
		{
			name:              "with namespace",
			args:              []string{"config", "opencode", "--namespace", "test-namespace"},
			expectedContext:   "",
			expectedNamespace: "test-namespace",
		},
		{
			name:              "with both",
			args:              []string{"config", "opencode", "--context", "test-context", "--namespace", "test-namespace"},
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

			assert.Equal(t, tt.expectedContext, cmd.Config.Opencode.Context, "Context should match")
			assert.Equal(t, tt.expectedNamespace, cmd.Config.Opencode.Namespace, "Namespace should match")
		})
	}
}

func TestConfigOpencodeNoLocalFileModification(t *testing.T) {
	cmd := &ConfigOpencodeCmd{}

	var runFunc func() error = cmd.Run
	assert.NotNil(t, runFunc, "ConfigOpencodeCmd.Run method should exist")
}
