package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoopCmdParsing covers the `ralph loop` command surface. It checks the
// optional slug argument, the repeatable --step flags, and the --max default
// of 10. It also checks the usage errors produced by Validate.
func TestLoopCmdParsing(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantSlug  string
		wantSteps []string
		wantMax   int
		wantErr   string
	}{
		{
			name:     "slug argument parses and max defaults to 10",
			args:     []string{"loop", "feature-x"},
			wantSlug: "feature-x",
			wantMax:  10,
		},
		{
			name:      "repeatable --step flags preserve order",
			args:      []string{"loop", "--step", "write code", "--step", "run tests"},
			wantSteps: []string{"write code", "run tests"},
			wantMax:   10,
		},
		{
			name:      "slug plus steps",
			args:      []string{"loop", "feature-x", "--step", "write code"},
			wantSlug:  "feature-x",
			wantSteps: []string{"write code"},
			wantMax:   10,
		},
		{
			name:     "explicit --max is parsed",
			args:     []string{"loop", "feature-x", "--max", "3"},
			wantSlug: "feature-x",
			wantMax:  3,
		},
		{
			name:    "usage error when neither slug nor step given",
			args:    []string{"loop"},
			wantErr: "a slug or at least one --step is required",
		},
		{
			name:    "zero --max rejected before execution",
			args:    []string{"loop", "feature-x", "--max", "0"},
			wantErr: "--max must be positive",
		},
		{
			name:    "negative --max rejected before execution",
			args:    []string{"loop", "feature-x", "--max=-1"},
			wantErr: "--max must be positive",
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
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSlug, cmd.Loop.Slug)
			assert.Equal(t, tt.wantSteps, cmd.Loop.Steps)
			assert.Equal(t, tt.wantMax, cmd.Loop.Max)
		})
	}
}

// TestLoopCmdHelpText asserts the loop subcommand's help appears even though
// Validate would otherwise fail. Kong prints help during its BeforeReset
// hook, which runs before Validate.
func TestLoopCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"loop", "--help"})
	assert.Contains(t, output, "Run AI iterations over a set of steps")
	assert.Contains(t, output, "--max")
	assert.Contains(t, output, "--step")
}

// TestLoopMaxNegativeSpaceFormRejected asserts kong rejects a negative --max
// in space form at parse time, before any execution begins.
func TestLoopMaxNegativeSpaceFormRejected(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"loop", "feature-x", "--max", "-1"})
	require.Error(t, err)
}

// writeLoopConfig writes a .ralph/config.yaml with the given loops section
// into the working directory.
func writeLoopConfig(t *testing.T, content string) {
	t.Helper()
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".ralph", "config.yaml"), []byte(content), 0644))
	t.Chdir(tmpDir)
}

// TestLoopRunWithMatchingSlug asserts Run resolves the matching loops: entry
// from the temp config and returns no error.
func TestLoopRunWithMatchingSlug(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
`)

	err := (&LoopCmd{Slug: "fmt"}).Run()
	require.NoError(t, err)
}

// TestLoopRunWithMissingSlug asserts Run returns an error carrying exactly
// "loop config not found: <slug>" when no loops: entry matches the slug.
func TestLoopRunWithMissingSlug(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	err := (&LoopCmd{Slug: "missing"}).Run()
	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: missing")
}

// TestLoopRunWithStepsWithoutSlug asserts Run accepts steps without a slug and
// needs no config file present.
func TestLoopRunWithStepsWithoutSlug(t *testing.T) {
	t.Chdir(t.TempDir())

	err := (&LoopCmd{Steps: []string{"run gofmt"}}).Run()
	require.NoError(t, err)
}
