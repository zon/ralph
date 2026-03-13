package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPulumiCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "config pulumi parses successfully with token",
			args:    []string{"config", "pulumi", "test-token"},
			wantErr: false,
		},
		{
			name:    "config pulumi parses successfully without token",
			args:    []string{"config", "pulumi"},
			wantErr: false,
		},
		{
			name:    "config pulumi with context",
			args:    []string{"config", "pulumi", "test-token", "--context", "my-cluster"},
			wantErr: false,
		},
		{
			name:    "config pulumi with namespace",
			args:    []string{"config", "pulumi", "test-token", "--namespace", "my-namespace"},
			wantErr: false,
		},
		{
			name:    "config pulumi with both context and namespace",
			args:    []string{"config", "pulumi", "test-token", "--context", "my-cluster", "--namespace", "my-namespace"},
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

func TestConfigPulumiFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedToken     string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "default values",
			args:              []string{"config", "pulumi", "test-token"},
			expectedToken:     "test-token",
			expectedContext:   "",
			expectedNamespace: "",
		},
		{
			name:              "with context",
			args:              []string{"config", "pulumi", "test-token", "--context", "test-context"},
			expectedToken:     "test-token",
			expectedContext:   "test-context",
			expectedNamespace: "",
		},
		{
			name:              "with namespace",
			args:              []string{"config", "pulumi", "test-token", "--namespace", "test-namespace"},
			expectedToken:     "test-token",
			expectedContext:   "",
			expectedNamespace: "test-namespace",
		},
		{
			name:              "with both",
			args:              []string{"config", "pulumi", "test-token", "--context", "test-context", "--namespace", "test-namespace"},
			expectedToken:     "test-token",
			expectedContext:   "test-context",
			expectedNamespace: "test-namespace",
		},
		{
			name:              "no token",
			args:              []string{"config", "pulumi"},
			expectedToken:     "",
			expectedContext:   "",
			expectedNamespace: "",
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

			assert.Equal(t, tt.expectedToken, cmd.Config.Pulumi.Token, "Token should match")
			assert.Equal(t, tt.expectedContext, cmd.Config.Pulumi.Context, "Context should match")
			assert.Equal(t, tt.expectedNamespace, cmd.Config.Pulumi.Namespace, "Namespace should match")
		})
	}
}
