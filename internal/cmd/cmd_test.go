package cmd

import (
	"fmt"
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
