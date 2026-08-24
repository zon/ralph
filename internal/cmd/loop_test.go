package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/ai"
)

// TestLoopCmdParsing covers the `ralph loop` command surface. It checks the
// optional slug argument, the repeatable --step flags, the --max default of
// 10, the --verbose flag, and the usage errors produced by Validate.
func TestLoopCmdParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantSlug    string
		wantSteps   []string
		wantMax     int
		wantVerbose bool
		wantErr     string
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
			name:        "explicit --verbose parses",
			args:        []string{"loop", "feature-x", "--verbose"},
			wantSlug:    "feature-x",
			wantMax:     10,
			wantVerbose: true,
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
			assert.Equal(t, tt.wantVerbose, cmd.Loop.Verbose)
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
	assert.Contains(t, output, "--verbose")
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

// TestLoopRunWithMatchingSlug asserts Run resolves the matching loops entry
// from the temp config, returns no error, and retains the resolved slug and
// steps on the command. The fake proposer guards against a regression where a
// given slug would consult the real AI.
func TestLoopRunWithMatchingSlug(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
`)

	proposer := &fakeSlugProposer{slug: "should-not-be-used"}
	cmd := &LoopCmd{Slug: "fmt", slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}}
	err := cmd.Run()
	require.NoError(t, err)
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is given")
	assert.Equal(t, "fmt", cmd.resolvedSlug, "the given slug is retained on the command")
	assert.Equal(t, []string{"run gofmt", "run go vet"}, cmd.resolvedSteps, "the config entry's steps are retained on the command")
}

// TestLoopRunWithMissingSlug asserts Run returns an error carrying exactly
// "loop config not found: <slug>" when no loops entry matches the slug. The
// fake proposer guards against a regression where a given slug would consult
// the real AI.
func TestLoopRunWithMissingSlug(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	proposer := &fakeSlugProposer{slug: "should-not-be-used"}
	err := (&LoopCmd{Slug: "missing", slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}}).Run()
	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is given")
}

// TestLoopRunWithStepsWithoutSlug asserts Run accepts steps without a slug and
// needs no config file present. The injected fake proposer supplies the slug,
// so the real AI (opencode) is never consulted. The proposed slug and the
// passed steps are retained on the command.
func TestLoopRunWithStepsWithoutSlug(t *testing.T) {
	t.Chdir(t.TempDir())

	proposer := &fakeSlugProposer{slug: "gofmt"}
	cmd := &LoopCmd{Steps: []string{"run gofmt"}, slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}}
	err := cmd.Run()
	require.NoError(t, err)
	assert.True(t, proposer.called, "the slug proposer is asked for a slug when none is given")
	assert.Equal(t, []string{"run gofmt"}, proposer.steps)
	assert.Equal(t, "gofmt", cmd.resolvedSlug, "the proposed slug is retained on the command")
	assert.Equal(t, []string{"run gofmt"}, cmd.resolvedSteps, "the passed steps are retained on the command")
}

// TestLoopRunWithStepsWithoutSlugPropagatesProposalError asserts that when the
// slug proposer fails (the "AI produces no usable slug" path), Run returns that
// error unchanged.
func TestLoopRunWithStepsWithoutSlugPropagatesProposalError(t *testing.T) {
	t.Chdir(t.TempDir())

	proposeErr := errors.New("no usable slug proposed by the AI")
	proposer := &fakeSlugProposer{err: proposeErr}
	err := (&LoopCmd{Steps: []string{"run gofmt"}, slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}}).Run()
	require.Error(t, err)
	assert.Equal(t, proposeErr, err)
	assert.True(t, proposer.called, "the slug proposer is consulted before failing")
}

// TestLoopRunWithSlugAndStepsUsesPassedSteps asserts the wired command prefers
// the passed steps over the config entry's steps. The config is present but
// its steps differ, and the slug is given, so the slug proposer must never be
// called. The given slug and the passed steps are retained on the command.
func TestLoopRunWithSlugAndStepsUsesPassedSteps(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	proposer := &fakeSlugProposer{slug: "should-not-be-used"}
	passed := []string{"write code", "run tests"}
	cmd := &LoopCmd{Slug: "fmt", Steps: passed, slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}}
	err := cmd.Run()
	require.NoError(t, err)
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is given")
	assert.Equal(t, "fmt", cmd.resolvedSlug, "the given slug is retained on the command")
	assert.Equal(t, passed, cmd.resolvedSteps, "the passed steps replace the config entry's steps on the command")
}

// fakeSlugProposer records the steps it was called with and returns an injected
// slug or error, so tests never invoke the real AI.
type fakeSlugProposer struct {
	steps  []string
	slug   string
	err    error
	called bool
}

func (f *fakeSlugProposer) ProposeSlug(steps []string) (string, error) {
	f.called = true
	f.steps = steps
	if f.err != nil {
		return "", f.err
	}
	return f.slug, nil
}

// fakeAIClient records the prompts it ran and returns an injected error when
// set, so tests never invoke the real AI.
type fakeAIClient struct {
	prompts []string
	err     error
	calls   int
}

func (f *fakeAIClient) RunAgent(prompt string) error {
	f.calls++
	f.prompts = append(f.prompts, prompt)
	return f.err
}

// fakeReportReader returns an injected report content or error, so tests never
// read the real report.md.
type fakeReportReader struct {
	content string
	err     error
}

func (f *fakeReportReader) ReadReport() (ai.Report, error) {
	if f.err != nil {
		return ai.Report{}, f.err
	}
	return ai.Report{Content: f.content}, nil
}

// TestLoopRunInvokesAIWithLoopPrompt asserts the wired command runs the built
// loop prompt through the injected AI client exactly once when the report says
// nothing to do. The given slug never consults the slug proposer.
func TestLoopRunInvokesAIWithLoopPrompt(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	ai := &fakeAIClient{}
	cmd := &LoopCmd{
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the report stops the loop")
	assert.Len(t, ai.prompts, 1)
	assert.Contains(t, ai.prompts[0], "run gofmt", "the loop prompt embeds the resolved steps")
}

// TestLoopRunStopsAfterMaxIterations asserts the wired command respects --max:
// a report that never says nothing to do runs exactly max AI passes.
func TestLoopRunStopsAfterMaxIterations(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	ai := &fakeAIClient{}
	cmd := &LoopCmd{
		Slug:         "fmt",
		Max:          3,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "did the work"},
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 3, ai.calls, "the AI runs exactly max iterations when the report never says nothing to do")
}

// TestLoopRunPropagatesAIError asserts an AI failure aborts the wired command
// and leaves the command without a resolved slug, because the loop fails before
// the resolution is retained.
func TestLoopRunPropagatesAIError(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	aiErr := errors.New("opencode execution failed: boom")
	ai := &fakeAIClient{err: aiErr}
	cmd := &LoopCmd{
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "did the work"},
	}
	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, aiErr, err, "the AI error is returned unchanged")
	assert.Empty(t, cmd.resolvedSlug, "no slug is retained when the loop fails")
}
