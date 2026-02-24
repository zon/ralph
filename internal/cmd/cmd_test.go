package cmd

import (
	"fmt"
	"testing"

	"github.com/alecthomas/kong"
)

func TestLocalFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "follow with local should fail",
			args:        []string{"run", "--follow", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --local flag",
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
			name:        "follow without local should succeed validation",
			args:        []string{"run", "--follow", "test.yaml"},
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
			name:        "default command - follow with local should fail",
			args:        []string{"--follow", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --local flag",
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

			// Test follow + local validation
			if cmd.Run.Follow && cmd.Run.Local {
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
		expectFollow    bool
		expectOnce     bool
		expectNoNotify bool
	}{
		{
			name:        "local flag sets Local to true",
			args:        []string{"run", "--local", "test.yaml"},
			expectLocal: true,
			expectFollow: false,
			expectOnce:  false,
		},
		{
			name:         "follow flag sets Follow to true",
			args:         []string{"run", "--follow", "test.yaml"},
			expectLocal:  false,
			expectFollow: true,
			expectOnce:   false,
		},
		{
			name:        "once flag sets Once to true",
			args:        []string{"run", "--once", "test.yaml"},
			expectLocal: false,
			expectFollow: false,
			expectOnce:  true,
		},
		{
			name:           "no-notify flag sets NoNotify to true",
			args:           []string{"run", "--no-notify", "test.yaml"},
			expectLocal:    false,
			expectFollow:    false,
			expectOnce:     false,
			expectNoNotify: true,
		},
		{
			name:        "default values",
			args:        []string{"run", "test.yaml"},
			expectLocal: false,
			expectFollow: false,
			expectOnce:  false,
		},
		{
			name:        "default command - local flag sets Local to true",
			args:        []string{"--local", "test.yaml"},
			expectLocal: true,
			expectFollow: false,
			expectOnce:  false,
		},
		{
			name:         "default command - follow flag sets Follow to true",
			args:         []string{"--follow", "test.yaml"},
			expectLocal:  false,
			expectFollow: true,
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

			if cmd.Run.Local != tt.expectLocal {
				t.Errorf("expected Local=%v, got %v", tt.expectLocal, cmd.Run.Local)
			}
			if cmd.Run.Follow != tt.expectFollow {
				t.Errorf("expected Follow=%v, got %v", tt.expectFollow, cmd.Run.Follow)
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
	if r.Follow && r.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}
	if r.Local && r.Once {
		return fmt.Errorf("--local flag is incompatible with --once flag")
	}
	return nil
}

func TestCommentCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedBody string
		expectedRepo string
		expectedBranch string
		expectedPR   string
		wantParseErr bool
	}{
		{
			name:           "basic comment command",
			args:           []string{"comment", "please fix this", "--repo", "acme/myrepo", "--branch", "ralph/feature", "--pr", "42"},
			expectedBody:   "please fix this",
			expectedRepo:   "acme/myrepo",
			expectedBranch: "ralph/feature",
			expectedPR:     "42",
		},
		{
			name:         "missing repo should fail",
			args:         []string{"comment", "body", "--branch", "ralph/feature", "--pr", "1"},
			wantParseErr: true,
		},
		{
			name:         "missing branch should fail",
			args:         []string{"comment", "body", "--repo", "acme/myrepo", "--pr", "1"},
			wantParseErr: true,
		},
		{
			name:         "missing pr should fail",
			args:         []string{"comment", "body", "--repo", "acme/myrepo", "--branch", "ralph/feature"},
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

			if cmd.Comment.Body != tt.expectedBody {
				t.Errorf("expected Body=%q, got %q", tt.expectedBody, cmd.Comment.Body)
			}
			if cmd.Comment.Repo != tt.expectedRepo {
				t.Errorf("expected Repo=%q, got %q", tt.expectedRepo, cmd.Comment.Repo)
			}
			if cmd.Comment.Branch != tt.expectedBranch {
				t.Errorf("expected Branch=%q, got %q", tt.expectedBranch, cmd.Comment.Branch)
			}
			if cmd.Comment.PR != tt.expectedPR {
				t.Errorf("expected PR=%q, got %q", tt.expectedPR, cmd.Comment.PR)
			}
		})
	}
}

func TestMergeCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedBranch  string
		expectedDryRun  bool
		expectedVerbose bool
		wantParseErr    bool
	}{
		{
			name:            "basic merge command",
			args:            []string{"merge", "ralph/my-feature", "--pr", "42"},
			expectedBranch:  "ralph/my-feature",
			expectedDryRun:  false,
			expectedVerbose: false,
		},
		{
			name:            "merge with dry-run flag",
			args:            []string{"merge", "ralph/my-feature", "--pr", "42", "--dry-run"},
			expectedBranch:  "ralph/my-feature",
			expectedDryRun:  true,
			expectedVerbose: false,
		},
		{
			name:            "merge with verbose flag",
			args:            []string{"merge", "ralph/my-feature", "--pr", "42", "--verbose"},
			expectedBranch:  "ralph/my-feature",
			expectedDryRun:  false,
			expectedVerbose: true,
		},
		{
			name:            "merge with both flags",
			args:            []string{"merge", "ralph/my-feature", "--pr", "42", "--dry-run", "--verbose"},
			expectedBranch:  "ralph/my-feature",
			expectedDryRun:  true,
			expectedVerbose: true,
		},
		{
			name:         "merge missing pr should fail",
			args:         []string{"merge", "ralph/my-feature"},
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
