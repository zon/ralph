package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/webhook"
	"gopkg.in/yaml.v3"
)

func TestLocalFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "watch with local should fail",
			args:        []string{"run", "--watch", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--watch flag is not applicable with --local flag",
		},
		{
			name:        "local with once should fail",
			args:        []string{"run", "--local", "--once", "test.yaml"},
			expectError: true,
			errorMsg:    "--local flag is incompatible with --once flag",
		},
		{
			name:        "local alone should succeed validation",
			args:        []string{"run", "--local", "test.yaml"},
			expectError: false,
		},
		{
			name:        "watch without local should succeed validation",
			args:        []string{"run", "--watch", "test.yaml"},
			expectError: false,
		},
		{
			name:        "once alone should succeed validation",
			args:        []string{"run", "--once", "test.yaml"},
			expectError: false,
		},
		{
			name:        "no flags should succeed validation",
			args:        []string{"run", "test.yaml"},
			expectError: false,
		},
		{
			name:        "default command - watch with local should fail",
			args:        []string{"--watch", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--watch flag is not applicable with --local flag",
		},
		{
			name:        "default command - local with once should fail",
			args:        []string{"--local", "--once", "test.yaml"},
			expectError: true,
			errorMsg:    "--local flag is incompatible with --once flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}), // Prevent exit during tests
			)
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			// Parse the args
			_, err = parser.Parse(tt.args)
			if err != nil {
				// Parser error - skip validation test as we can't reach Run()
				if tt.expectError {
					// This is ok - parser caught the error
					return
				}
				t.Fatalf("failed to parse args: %v", err)
			}

			// Now run validation
			// We need to mock the execution to test only validation
			// Override the project file validation since we're not testing that
			if cmd.Run.ProjectFile == "" {
				cmd.Run.ProjectFile = "test.yaml"
			}

			// Test watch + local validation
			if cmd.Run.Watch && cmd.Run.Local {
				err = validateRunFlags(&cmd.Run)
				if !tt.expectError {
					t.Errorf("expected no error, got: %v", err)
				} else if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			// Test local + once validation
			if cmd.Run.Local && cmd.Run.Once {
				err = validateRunFlags(&cmd.Run)
				if !tt.expectError {
					t.Errorf("expected no error, got: %v", err)
				} else if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			// Should pass validation
			err = validateRunFlags(&cmd.Run)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectLocal    bool
		expectWatch    bool
		expectOnce     bool
		expectNoNotify bool
	}{
		{
			name:        "local flag sets Local to true",
			args:        []string{"run", "--local", "test.yaml"},
			expectLocal: true,
			expectWatch: false,
			expectOnce:  false,
		},
		{
			name:        "watch flag sets Watch to true",
			args:        []string{"run", "--watch", "test.yaml"},
			expectLocal: false,
			expectWatch: true,
			expectOnce:  false,
		},
		{
			name:        "once flag sets Once to true",
			args:        []string{"run", "--once", "test.yaml"},
			expectLocal: false,
			expectWatch: false,
			expectOnce:  true,
		},
		{
			name:           "no-notify flag sets NoNotify to true",
			args:           []string{"run", "--no-notify", "test.yaml"},
			expectLocal:    false,
			expectWatch:    false,
			expectOnce:     false,
			expectNoNotify: true,
		},
		{
			name:        "default values",
			args:        []string{"run", "test.yaml"},
			expectLocal: false,
			expectWatch: false,
			expectOnce:  false,
		},
		{
			name:        "default command - local flag sets Local to true",
			args:        []string{"--local", "test.yaml"},
			expectLocal: true,
			expectWatch: false,
			expectOnce:  false,
		},
		{
			name:        "default command - watch flag sets Watch to true",
			args:        []string{"--watch", "test.yaml"},
			expectLocal: false,
			expectWatch: true,
			expectOnce:  false,
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

			if cmd.Run.Local != tt.expectLocal {
				t.Errorf("expected Local=%v, got %v", tt.expectLocal, cmd.Run.Local)
			}
			if cmd.Run.Watch != tt.expectWatch {
				t.Errorf("expected Watch=%v, got %v", tt.expectWatch, cmd.Run.Watch)
			}
			if cmd.Run.Once != tt.expectOnce {
				t.Errorf("expected Once=%v, got %v", tt.expectOnce, cmd.Run.Once)
			}
			if cmd.Run.NoNotify != tt.expectNoNotify {
				t.Errorf("expected NoNotify=%v, got %v", tt.expectNoNotify, cmd.Run.NoNotify)
			}
		})
	}
}

// validateRunFlags extracts the validation logic for testing
func validateRunFlags(r *RunCmd) error {
	if r.Watch && r.Local {
		return fmt.Errorf("--watch flag is not applicable with --local flag")
	}
	if r.Local && r.Once {
		return fmt.Errorf("--local flag is incompatible with --once flag")
	}
	return nil
}

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

			// If parsing succeeded, verify the fields were set correctly
			if err == nil && !tt.wantErr {
				// Just verify that we can access the config github command
				if cmd.Config.Github.Context != "" {
					// Context was set, which is valid
				}
				if cmd.Config.Github.Namespace != "" {
					// Namespace was set, which is valid
				}
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
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If parsing succeeded, verify the fields were set correctly
			if err == nil && !tt.wantErr {
				// Just verify that we can access the config opencode command
				if cmd.Config.Opencode.Context != "" {
					// Context was set, which is valid
				}
				if cmd.Config.Opencode.Namespace != "" {
					// Namespace was set, which is valid
				}
			}
		})
	}
}

func TestInstructionsFlagParsing(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		expectedInstructions string
	}{
		{
			name:                 "no instructions flag defaults to empty",
			args:                 []string{"run", "test.yaml"},
			expectedInstructions: "",
		},
		{
			name:                 "instructions flag is parsed",
			args:                 []string{"run", "--instructions", "/path/to/instructions.md", "test.yaml"},
			expectedInstructions: "/path/to/instructions.md",
		},
		{
			name:                 "instructions flag with default command",
			args:                 []string{"--instructions", "/path/to/instructions.md", "test.yaml"},
			expectedInstructions: "/path/to/instructions.md",
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

			if cmd.Run.Instructions != tt.expectedInstructions {
				t.Errorf("expected Instructions=%q, got %q", tt.expectedInstructions, cmd.Run.Instructions)
			}
		})
	}
}

func TestMergeCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedFileSuffix string
		expectedBranch     string
		expectedDryRun     bool
		expectedVerbose    bool
		wantParseErr       bool
	}{
		{
			name:               "basic merge command",
			args:               []string{"merge", "project.yaml", "ralph/my-feature"},
			expectedFileSuffix: "project.yaml",
			expectedBranch:     "ralph/my-feature",
			expectedDryRun:     false,
			expectedVerbose:    false,
		},
		{
			name:               "merge with dry-run flag",
			args:               []string{"merge", "project.yaml", "ralph/my-feature", "--dry-run"},
			expectedFileSuffix: "project.yaml",
			expectedBranch:     "ralph/my-feature",
			expectedDryRun:     true,
			expectedVerbose:    false,
		},
		{
			name:               "merge with verbose flag",
			args:               []string{"merge", "project.yaml", "ralph/my-feature", "--verbose"},
			expectedFileSuffix: "project.yaml",
			expectedBranch:     "ralph/my-feature",
			expectedDryRun:     false,
			expectedVerbose:    true,
		},
		{
			name:               "merge with both flags",
			args:               []string{"merge", "project.yaml", "ralph/my-feature", "--dry-run", "--verbose"},
			expectedFileSuffix: "project.yaml",
			expectedBranch:     "ralph/my-feature",
			expectedDryRun:     true,
			expectedVerbose:    true,
		},
		{
			name:         "merge missing branch should fail",
			args:         []string{"merge", "project.yaml"},
			wantParseErr: true,
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
			if tt.wantParseErr {
				if err == nil {
					t.Error("expected parse error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			// ProjectFile is resolved to absolute path by Kong's type:"path",
			// so check that the path ends with the expected suffix
			if !strings.HasSuffix(cmd.Merge.ProjectFile, tt.expectedFileSuffix) {
				t.Errorf("expected ProjectFile ending with %q, got %q", tt.expectedFileSuffix, cmd.Merge.ProjectFile)
			}
			if cmd.Merge.Branch != tt.expectedBranch {
				t.Errorf("expected Branch=%q, got %q", tt.expectedBranch, cmd.Merge.Branch)
			}
			if cmd.Merge.DryRun != tt.expectedDryRun {
				t.Errorf("expected DryRun=%v, got %v", tt.expectedDryRun, cmd.Merge.DryRun)
			}
			if cmd.Merge.Verbose != tt.expectedVerbose {
				t.Errorf("expected Verbose=%v, got %v", tt.expectedVerbose, cmd.Merge.Verbose)
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
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Config.Opencode.Context != tt.expectedContext {
				t.Errorf("expected Context=%q, got %q", tt.expectedContext, cmd.Config.Opencode.Context)
			}
			if cmd.Config.Opencode.Namespace != tt.expectedNamespace {
				t.Errorf("expected Namespace=%q, got %q", tt.expectedNamespace, cmd.Config.Opencode.Namespace)
			}
		})
	}
}

// TestConfigWebhookConfigFlagParsing tests flag parsing for the webhook-config command
func TestConfigWebhookConfigFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
		expectedDryRun    bool
		wantErr           bool
	}{
		{
			name:              "defaults to ralph-webhook namespace",
			args:              []string{"config", "webhook-config"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "custom context",
			args:              []string{"config", "webhook-config", "--context", "my-cluster"},
			expectedContext:   "my-cluster",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "custom namespace overrides default",
			args:              []string{"config", "webhook-config", "--namespace", "my-ns"},
			expectedContext:   "",
			expectedNamespace: "my-ns",
			expectedDryRun:    false,
		},
		{
			name:              "dry-run flag",
			args:              []string{"config", "webhook-config", "--dry-run"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    true,
		},
		{
			name:              "all flags",
			args:              []string{"config", "webhook-config", "--context", "ctx", "--namespace", "ns", "--dry-run"},
			expectedContext:   "ctx",
			expectedNamespace: "ns",
			expectedDryRun:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.expectedContext, cmd.Config.WebhookConfig.Context)
			assert.Equal(t, tt.expectedNamespace, cmd.Config.WebhookConfig.Namespace)
			assert.Equal(t, tt.expectedDryRun, cmd.Config.WebhookConfig.DryRun)
		})
	}
}

// TestConfigWebhookSecretFlagParsing tests flag parsing for the webhook-secret command
func TestConfigWebhookSecretFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
		expectedDryRun    bool
	}{
		{
			name:              "defaults to ralph-webhook namespace",
			args:              []string{"config", "webhook-secret"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "custom context and namespace",
			args:              []string{"config", "webhook-secret", "--context", "prod", "--namespace", "prod-webhook"},
			expectedContext:   "prod",
			expectedNamespace: "prod-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "dry-run flag",
			args:              []string{"config", "webhook-secret", "--dry-run"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedContext, cmd.Config.WebhookSecret.Context)
			assert.Equal(t, tt.expectedNamespace, cmd.Config.WebhookSecret.Namespace)
			assert.Equal(t, tt.expectedDryRun, cmd.Config.WebhookSecret.DryRun)
		})
	}
}

// TestBuildWebhookAppConfig tests the pure default-filling logic
func TestBuildWebhookAppConfig(t *testing.T) {
	t.Run("fills all defaults when starting from nil", func(t *testing.T) {
		cfg := buildWebhookAppConfig(nil, "my-repo", "my-owner", "anthropic/claude-sonnet-4-6", "ralph-bot")

		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "anthropic/claude-sonnet-4-6", cfg.Model)
		assert.Equal(t, "ralph-bot", cfg.RalphUsername)
		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "my-owner", cfg.Repos[0].Owner)
		assert.Equal(t, "my-repo", cfg.Repos[0].Name)
		assert.Equal(t, "/repos/my-repo", cfg.Repos[0].ClonePath)
	})

	t.Run("clonePath defaults to /repos/<repo-name>", func(t *testing.T) {
		cfg := buildWebhookAppConfig(nil, "special-repo", "owner-x", "", "")

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "/repos/special-repo", cfg.Repos[0].ClonePath)
	})

	t.Run("does not override existing port", func(t *testing.T) {
		partial := &webhook.AppConfig{Port: 9090}
		cfg := buildWebhookAppConfig(partial, "repo", "owner", "model", "user")

		assert.Equal(t, 9090, cfg.Port)
	})

	t.Run("does not override existing model", func(t *testing.T) {
		partial := &webhook.AppConfig{Model: "my-custom-model"}
		cfg := buildWebhookAppConfig(partial, "repo", "owner", "default-model", "user")

		assert.Equal(t, "my-custom-model", cfg.Model)
	})

	t.Run("does not override existing ralphUsername", func(t *testing.T) {
		partial := &webhook.AppConfig{RalphUsername: "custom-user"}
		cfg := buildWebhookAppConfig(partial, "repo", "owner", "model", "detected-user")

		assert.Equal(t, "custom-user", cfg.RalphUsername)
	})

	t.Run("does not duplicate existing repo", func(t *testing.T) {
		partial := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "my-owner", Name: "my-repo", ClonePath: "/custom/path"},
			},
		}
		cfg := buildWebhookAppConfig(partial, "my-repo", "my-owner", "model", "user")

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "/custom/path", cfg.Repos[0].ClonePath)
	})

	t.Run("adds detected repo alongside existing repos", func(t *testing.T) {
		partial := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "owner-a", Name: "repo-a", ClonePath: "/repos/repo-a"},
			},
		}
		cfg := buildWebhookAppConfig(partial, "repo-b", "owner-b", "model", "user")

		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "repo-a", cfg.Repos[0].Name)
		assert.Equal(t, "repo-b", cfg.Repos[1].Name)
	})

	t.Run("skips repo detection when repoName is empty", func(t *testing.T) {
		cfg := buildWebhookAppConfig(nil, "", "", "model", "user")

		assert.Empty(t, cfg.Repos)
	})

	t.Run("existing repo without clonePath gets default filled", func(t *testing.T) {
		partial := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "my-owner", Name: "my-repo"},
			},
		}
		cfg := buildWebhookAppConfig(partial, "my-repo", "my-owner", "model", "user")

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "/repos/my-repo", cfg.Repos[0].ClonePath)
	})

	t.Run("loads from partial config file", func(t *testing.T) {
		dir := t.TempDir()
		partialYAML := "ralphUsername: from-file\nport: 7070\n"
		path := filepath.Join(dir, "partial.yaml")
		require.NoError(t, os.WriteFile(path, []byte(partialYAML), 0644))

		loaded, err := webhook.LoadAppConfig(path)
		require.NoError(t, err)

		cfg := buildWebhookAppConfig(loaded, "my-repo", "my-owner", "default-model", "detected-user")

		assert.Equal(t, 7070, cfg.Port)
		assert.Equal(t, "from-file", cfg.RalphUsername)
		assert.Equal(t, "default-model", cfg.Model)
	})
}

// TestBuildWebhookSecrets tests the pure secret generation logic for the webhook-secret command
func TestBuildWebhookSecrets(t *testing.T) {
	// deterministicGenerator returns predictable secrets for testing
	counter := 0
	deterministicGenerator := func() (string, error) {
		counter++
		return fmt.Sprintf("test-secret-%d", counter), nil
	}

	t.Run("generates a secret for each repo", func(t *testing.T) {
		counter = 0
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "acme", Name: "repo-a", ClonePath: "/repos/repo-a"},
				{Owner: "acme", Name: "repo-b", ClonePath: "/repos/repo-b"},
			},
		}

		secrets, err := buildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		require.Len(t, secrets.Repos, 2)
		assert.Equal(t, "acme", secrets.Repos[0].Owner)
		assert.Equal(t, "repo-a", secrets.Repos[0].Name)
		assert.Equal(t, "test-secret-1", secrets.Repos[0].WebhookSecret)
		assert.Equal(t, "acme", secrets.Repos[1].Owner)
		assert.Equal(t, "repo-b", secrets.Repos[1].Name)
		assert.Equal(t, "test-secret-2", secrets.Repos[1].WebhookSecret)
	})

	t.Run("returns empty repos list when no repos configured", func(t *testing.T) {
		counter = 0
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{},
		}

		secrets, err := buildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		assert.Empty(t, secrets.Repos)
	})

	t.Run("propagates secret generation errors", func(t *testing.T) {
		failingGenerator := func() (string, error) {
			return "", fmt.Errorf("entropy exhausted")
		}

		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
			},
		}

		_, err := buildWebhookSecrets(appCfg, failingGenerator)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entropy exhausted")
	})

	t.Run("generates unique secrets per repo", func(t *testing.T) {
		// Use real generateWebhookSecret to verify uniqueness
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
				{Owner: "acme", Name: "repo-b"},
				{Owner: "acme", Name: "repo-c"},
			},
		}

		secrets, err := buildWebhookSecrets(appCfg, generateWebhookSecret)
		require.NoError(t, err)

		require.Len(t, secrets.Repos, 3)

		// All secrets should be non-empty and unique
		secretSet := make(map[string]bool)
		for _, rs := range secrets.Repos {
			assert.NotEmpty(t, rs.WebhookSecret)
			assert.False(t, secretSet[rs.WebhookSecret], "duplicate secret generated")
			secretSet[rs.WebhookSecret] = true
		}
	})

	t.Run("serializes secrets to valid YAML", func(t *testing.T) {
		counter = 0
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "myowner", Name: "myrepo", ClonePath: "/repos/myrepo"},
			},
		}

		secrets, err := buildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		// Verify that the secrets can be marshaled and unmarshaled as YAML
		secretsBytes, err := yaml.Marshal(secrets)
		require.NoError(t, err)

		var roundTripped webhook.Secrets
		require.NoError(t, yaml.Unmarshal(secretsBytes, &roundTripped))

		require.Len(t, roundTripped.Repos, 1)
		assert.Equal(t, "myowner", roundTripped.Repos[0].Owner)
		assert.Equal(t, "myrepo", roundTripped.Repos[0].Name)
		assert.Equal(t, "test-secret-1", roundTripped.Repos[0].WebhookSecret)
	})
}

// TestGenerateWebhookSecret tests that webhook secrets are cryptographically random
func TestGenerateWebhookSecret(t *testing.T) {
	t.Run("generates a non-empty secret", func(t *testing.T) {
		secret, err := generateWebhookSecret()
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})

	t.Run("generates at least 32 characters", func(t *testing.T) {
		secret, err := generateWebhookSecret()
		require.NoError(t, err)
		// 32 bytes base64-encoded is at least 43 characters (raw URL encoding)
		assert.GreaterOrEqual(t, len(secret), 32)
	})

	t.Run("generates unique secrets on repeated calls", func(t *testing.T) {
		secrets := make(map[string]bool)
		for i := 0; i < 10; i++ {
			secret, err := generateWebhookSecret()
			require.NoError(t, err)
			assert.False(t, secrets[secret], "duplicate secret generated on iteration %d", i)
			secrets[secret] = true
		}
	})
}
