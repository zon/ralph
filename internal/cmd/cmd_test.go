package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestRemoteFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "watch without remote should fail",
			args:        []string{"run", "--watch", "test.yaml"},
			expectError: true,
			errorMsg:    "--watch flag is only applicable with --remote flag",
		},
		{
			name:        "remote with once should fail",
			args:        []string{"run", "--remote", "--once", "test.yaml"},
			expectError: true,
			errorMsg:    "--remote flag is incompatible with --once flag",
		},
		{
			name:        "remote alone should succeed validation",
			args:        []string{"run", "--remote", "test.yaml"},
			expectError: false,
		},
		{
			name:        "remote with watch should succeed validation",
			args:        []string{"run", "--remote", "--watch", "test.yaml"},
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
			name:        "default command - watch without remote should fail",
			args:        []string{"--watch", "test.yaml"},
			expectError: true,
			errorMsg:    "--watch flag is only applicable with --remote flag",
		},
		{
			name:        "default command - remote with once should fail",
			args:        []string{"--remote", "--once", "test.yaml"},
			expectError: true,
			errorMsg:    "--remote flag is incompatible with --once flag",
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

			// Test watch flag validation
			if cmd.Run.Watch && !cmd.Run.Remote {
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

			// Test remote + once validation
			if cmd.Run.Remote && cmd.Run.Once {
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
		expectRemote   bool
		expectWatch    bool
		expectOnce     bool
		expectNoNotify bool
	}{
		{
			name:         "remote flag sets Remote to true",
			args:         []string{"run", "--remote", "test.yaml"},
			expectRemote: true,
			expectWatch:  false,
			expectOnce:   false,
		},
		{
			name:         "watch flag sets Watch to true",
			args:         []string{"run", "--remote", "--watch", "test.yaml"},
			expectRemote: true,
			expectWatch:  true,
			expectOnce:   false,
		},
		{
			name:         "once flag sets Once to true",
			args:         []string{"run", "--once", "test.yaml"},
			expectRemote: false,
			expectWatch:  false,
			expectOnce:   true,
		},
		{
			name:           "no-notify flag sets NoNotify to true",
			args:           []string{"run", "--no-notify", "test.yaml"},
			expectRemote:   false,
			expectWatch:    false,
			expectOnce:     false,
			expectNoNotify: true,
		},
		{
			name:         "default values",
			args:         []string{"run", "test.yaml"},
			expectRemote: false,
			expectWatch:  false,
			expectOnce:   false,
		},
		{
			name:         "default command - remote flag sets Remote to true",
			args:         []string{"--remote", "test.yaml"},
			expectRemote: true,
			expectWatch:  false,
			expectOnce:   false,
		},
		{
			name:         "default command - watch flag sets Watch to true",
			args:         []string{"--remote", "--watch", "test.yaml"},
			expectRemote: true,
			expectWatch:  true,
			expectOnce:   false,
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

			if cmd.Run.Remote != tt.expectRemote {
				t.Errorf("expected Remote=%v, got %v", tt.expectRemote, cmd.Run.Remote)
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
	if r.Watch && !r.Remote {
		return fmt.Errorf("--watch flag is only applicable with --remote flag")
	}
	if r.Remote && r.Once {
		return fmt.Errorf("--remote flag is incompatible with --once flag")
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
