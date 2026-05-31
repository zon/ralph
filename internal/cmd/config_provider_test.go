package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigProviderCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "config provider parses successfully",
			args:    []string{"config", "provider", "anthropic"},
			wantErr: false,
		},
		{
			name:    "config provider with context",
			args:    []string{"config", "provider", "google", "--context", "my-cluster"},
			wantErr: false,
		},
		{
			name:    "config provider with namespace",
			args:    []string{"config", "provider", "deepseek", "--namespace", "my-namespace"},
			wantErr: false,
		},
		{
			name:    "config provider with both context and namespace",
			args:    []string{"config", "provider", "anthropic", "--context", "my-cluster", "--namespace", "my-namespace"},
			wantErr: false,
		},
		{
			name:    "missing provider should fail",
			args:    []string{"config", "provider"},
			wantErr: true,
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

func TestConfigProviderFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedProvider  string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "default values",
			args:              []string{"config", "provider", "anthropic"},
			expectedProvider:  "anthropic",
			expectedContext:   "",
			expectedNamespace: "",
		},
		{
			name:              "with context",
			args:              []string{"config", "provider", "google", "--context", "test-context"},
			expectedProvider:  "google",
			expectedContext:   "test-context",
			expectedNamespace: "",
		},
		{
			name:              "with namespace",
			args:              []string{"config", "provider", "deepseek", "--namespace", "test-namespace"},
			expectedProvider:  "deepseek",
			expectedContext:   "",
			expectedNamespace: "test-namespace",
		},
		{
			name:              "with both",
			args:              []string{"config", "provider", "anthropic", "--context", "test-context", "--namespace", "test-namespace"},
			expectedProvider:  "anthropic",
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

			assert.Equal(t, tt.expectedProvider, cmd.Config.Provider.Provider, "Provider should match")
			assert.Equal(t, tt.expectedContext, cmd.Config.Provider.Context, "Context should match")
			assert.Equal(t, tt.expectedNamespace, cmd.Config.Provider.Namespace, "Namespace should match")
		})
	}
}

func TestConfigProviderRun_ValidatesProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  string
	}{
		{
			name:     "anthropic is valid",
			provider: "anthropic",
			wantErr:  "",
		},
		{
			name:     "google is valid",
			provider: "google",
			wantErr:  "",
		},
		{
			name:     "deepseek is valid",
			provider: "deepseek",
			wantErr:  "",
		},
		{
			name:     "unknown provider returns error",
			provider: "openai",
			wantErr:  "unknown provider: openai",
		},
		{
			name:     "empty provider returns error",
			provider: "",
			wantErr:  "unknown provider: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ConfigProviderCmd{Provider: tt.provider}
			err := cmd.Run()
			if tt.wantErr == "" {
				// For valid providers, Run() will fail on config loading or k8s,
				// but NOT with an "unknown provider" error
				assert.Error(t, err, "expected some error but not unknown provider")
				assert.NotContains(t, err.Error(), "unknown provider")
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
