package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
)

func TestConfigGithubCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "config github parses successfully",
			args:    []string{"config", "github"},
			wantErr: false,
		},
		{
			name:    "config github with context",
			args:    []string{"config", "github", "--context", "my-cluster"},
			wantErr: false,
		},
		{
			name:    "config github with namespace",
			args:    []string{"config", "github", "--namespace", "my-namespace"},
			wantErr: false,
		},
		{
			name:    "config github with both context and namespace",
			args:    []string{"config", "github", "--context", "my-cluster", "--namespace", "my-namespace"},
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
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigGithubFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "default values",
			args:              []string{"config", "github"},
			expectedContext:   "",
			expectedNamespace: "",
		},
		{
			name:              "with context",
			args:              []string{"config", "github", "--context", "test-context"},
			expectedContext:   "test-context",
			expectedNamespace: "",
		},
		{
			name:              "with namespace",
			args:              []string{"config", "github", "--namespace", "test-namespace"},
			expectedContext:   "",
			expectedNamespace: "test-namespace",
		},
		{
			name:              "with both",
			args:              []string{"config", "github", "--context", "test-context", "--namespace", "test-namespace"},
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
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Config.Github.Context != tt.expectedContext {
				t.Errorf("expected Context=%q, got %q", tt.expectedContext, cmd.Config.Github.Context)
			}
			if cmd.Config.Github.Namespace != tt.expectedNamespace {
				t.Errorf("expected Namespace=%q, got %q", tt.expectedNamespace, cmd.Config.Github.Namespace)
			}
		})
	}
}
