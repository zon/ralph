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
// 10, the --verbose flag, the --local and --follow flags, and the usage errors
// produced by Validate.
func TestLoopCmdParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantSlug    string
		wantSteps   []string
		wantMax     int
		wantVerbose bool
		wantLocal   bool
		wantFollow  bool
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
			name:      "--local parses",
			args:      []string{"loop", "feature-x", "--local"},
			wantSlug:  "feature-x",
			wantMax:   10,
			wantLocal: true,
		},
		{
			name:       "--follow parses",
			args:       []string{"loop", "feature-x", "--follow"},
			wantSlug:   "feature-x",
			wantMax:    10,
			wantFollow: true,
		},
		{
			name:       "-f short form parses",
			args:       []string{"loop", "feature-x", "-f"},
			wantSlug:   "feature-x",
			wantMax:    10,
			wantFollow: true,
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
		{
			name:    "--follow rejected together with --local before execution",
			args:    []string{"loop", "feature-x", "--local", "--follow"},
			wantErr: "--follow flag is not applicable with --local flag",
		},
		{
			name:    "--follow rejected together with --local in short form before execution",
			args:    []string{"loop", "feature-x", "--local", "-f"},
			wantErr: "--follow flag is not applicable with --local flag",
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
			assert.Equal(t, tt.wantLocal, cmd.Loop.Local)
			assert.Equal(t, tt.wantFollow, cmd.Loop.Follow)
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
	assert.Contains(t, output, "--local")
	assert.Contains(t, output, "--follow")
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
	cmd := &LoopCmd{Local: true, Slug: "fmt", slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}, prClient: &fakePullRequestOpener{}}
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
	err := (&LoopCmd{Local: true, Slug: "missing", slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}, prClient: &fakePullRequestOpener{}}).Run()
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
	cmd := &LoopCmd{Local: true, Steps: []string{"run gofmt"}, slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}, prClient: &fakePullRequestOpener{}}
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
	err := (&LoopCmd{Local: true, Steps: []string{"run gofmt"}, slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}, prClient: &fakePullRequestOpener{}}).Run()
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
	cmd := &LoopCmd{Local: true, Slug: "fmt", Steps: passed, slugProposer: proposer, aiClient: &fakeAIClient{}, reportReader: &fakeReportReader{content: "NOTHING_TO_DO"}, prClient: &fakePullRequestOpener{}}
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

// fakeGitClient records the slugs it committed and returns an injected error
// when set, so tests never touch a real git repository.
type fakeGitClient struct {
	slugs []string
	err   error
	calls int
}

func (f *fakeGitClient) CommitIterationAndPush(slug string) error {
	f.calls++
	f.slugs = append(f.slugs, slug)
	return f.err
}

// fakePullRequestOpener records the slugs it opened pull requests for and
// returns an injected error when set, so tests never touch the real GitHub
// client.
type fakePullRequestOpener struct {
	slugs []string
	err   error
	calls int
}

func (f *fakePullRequestOpener) OpenLoopPullRequest(slug string) error {
	f.calls++
	f.slugs = append(f.slugs, slug)
	return f.err
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
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
		prClient:     &fakePullRequestOpener{},
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
	git := &fakeGitClient{}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          3,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "did the work"},
		gitClient:    git,
		prClient:     &fakePullRequestOpener{},
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 3, ai.calls, "the AI runs exactly max iterations when the report never says nothing to do")
	assert.Equal(t, 3, git.calls, "each non-nothing-to-do iteration is committed exactly once")
	require.Len(t, git.slugs, 3, "every commit records the slug it was called with")
	for _, slug := range git.slugs {
		assert.Equal(t, "fmt", slug, "every iteration commits the resolved slug")
	}
}

// TestLoopRunNothingToDoDoesNotCommit asserts an iteration whose report says
// nothing to do runs the AI once but is not committed: the git client is never
// called. The resolved slug is still retained on the command for the later loop
// phases.
func TestLoopRunNothingToDoDoesNotCommit(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	ai := &fakeAIClient{}
	git := &fakeGitClient{}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
		gitClient:    git,
		prClient:     &fakePullRequestOpener{},
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the report stops the loop")
	assert.Equal(t, 0, git.calls, "a nothing-to-do iteration is not committed")
	assert.Empty(t, git.slugs, "no iteration commits a slug when there is nothing to do")
	assert.Equal(t, "fmt", cmd.resolvedSlug, "the resolved slug is retained on the command")
}

// TestLoopRunPropagatesIterationCommitError asserts a commit failure aborts
// the wired command: the error is returned unchanged and no resolved slug is
// retained on the command.
func TestLoopRunPropagatesIterationCommitError(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	commitErr := errors.New("failed to push loop-fmt: boom")
	git := &fakeGitClient{err: commitErr}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     &fakeAIClient{},
		reportReader: &fakeReportReader{content: "did the work"},
		gitClient:    git,
		prClient:     &fakePullRequestOpener{},
	}
	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, commitErr, err, "the iteration commit error is returned unchanged")
	assert.Empty(t, cmd.resolvedSlug, "no slug is retained when the iteration commit fails")
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
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "did the work"},
		prClient:     &fakePullRequestOpener{},
	}
	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, aiErr, err, "the AI error is returned unchanged")
	assert.Empty(t, cmd.resolvedSlug, "no slug is retained when the loop fails")
}

// TestLoopRunOpensPullRequestAfterCommits asserts the wired command opens the
// loop branch's pull request once for the resolved slug after the loop commits
// its work.
func TestLoopRunOpensPullRequestAfterCommits(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	ai := &fakeAIClient{}
	git := &fakeGitClient{}
	pr := &fakePullRequestOpener{}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          1,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     ai,
		reportReader: &fakeReportReader{content: "did the work"},
		gitClient:    git,
		prClient:     pr,
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 1, git.calls, "the work iteration is committed once")
	assert.Equal(t, []string{"fmt"}, git.slugs, "the commit records the resolved slug")
	assert.Equal(t, 1, pr.calls, "the pull request is opened exactly once after the loop ends")
	assert.Equal(t, []string{"fmt"}, pr.slugs, "the pull request is opened for the resolved slug")
	assert.Equal(t, "fmt", cmd.resolvedSlug, "the resolved slug is retained on the command")
}

// TestLoopRunDelegatesPullRequestWhenNothingCommitted asserts the wired command
// still delegates opening the pull request when no iteration committed work:
// the git client is never called, yet the pull request opener is called once
// with the resolved slug, because the implementation decides nothing was
// committed.
func TestLoopRunDelegatesPullRequestWhenNothingCommitted(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	git := &fakeGitClient{}
	pr := &fakePullRequestOpener{}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     &fakeAIClient{},
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
		gitClient:    git,
		prClient:     pr,
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, git.calls, "a nothing-to-do iteration is not committed")
	assert.Equal(t, 1, pr.calls, "the pull request open is delegated exactly once after the loop ends")
	assert.Equal(t, []string{"fmt"}, pr.slugs, "the pull request open receives the resolved slug")
	assert.Equal(t, "fmt", cmd.resolvedSlug, "the resolved slug is retained on the command")
}

// TestLoopRunPropagatesPullRequestOpenError asserts a pull request open failure
// after the loop ends aborts the wired command and is returned unchanged.
func TestLoopRunPropagatesPullRequestOpenError(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	prErr := errors.New("failed to open loop pull request: boom")
	pr := &fakePullRequestOpener{err: prErr}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     &fakeAIClient{},
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
		prClient:     pr,
	}
	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, prErr, err, "the pull request open error is returned unchanged")
	assert.Equal(t, 1, pr.calls, "the pull request open is attempted once after the loop ends")
}

// fakeLoopRemoteRunner records the invocation and returns an injected error
// when set, so tests never touch the real argo client.
type fakeLoopRemoteRunner struct {
	slug   string
	steps  []string
	max    int
	err    error
	called bool
}

func (f *fakeLoopRemoteRunner) Run(slug string, steps []string, max int) error {
	f.called = true
	f.slug = slug
	f.steps = steps
	f.max = max
	return f.err
}

// TestLoopRunRemoteSubmitsWorkflow asserts the default (without --local) run
// path delegates to the remote runner, which submits the loop workflow carrying
// the slug, steps, and max iterations. No loop config is needed because the
// remote path never runs the loop in-process.
func TestLoopRunRemoteSubmitsWorkflow(t *testing.T) {
	t.Chdir(t.TempDir())

	runner := &fakeLoopRemoteRunner{}
	cmd := &LoopCmd{Slug: "fmt", Steps: []string{"run gofmt"}, Max: 3, remoteRunner: runner}
	err := cmd.Run()
	require.NoError(t, err)
	assert.True(t, runner.called, "the remote runner is consulted without --local")
	assert.Equal(t, "fmt", runner.slug, "the remote runner receives the slug")
	assert.Equal(t, []string{"run gofmt"}, runner.steps, "the remote runner receives the steps")
	assert.Equal(t, 3, runner.max, "the remote runner receives the max iterations")
	assert.Empty(t, cmd.resolvedSlug, "no slug is retained in local-mode fields on the remote path")
}

// TestLoopRunRemotePropagatesSubmitError asserts a workflow submission failure
// aborts the default run path and is returned unchanged.
func TestLoopRunRemotePropagatesSubmitError(t *testing.T) {
	t.Chdir(t.TempDir())

	submitErr := errors.New("failed to submit workflow: boom")
	runner := &fakeLoopRemoteRunner{err: submitErr}
	cmd := &LoopCmd{Slug: "fmt", Max: 10, remoteRunner: runner}
	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, submitErr, err, "the workflow submission error is returned unchanged")
	assert.True(t, runner.called, "the remote runner is consulted before failing")
}

// TestLoopRunLocalDoesNotSubmitWorkflow asserts --local runs the loop in-process
// and never consults the remote runner, so no workflow is submitted.
func TestLoopRunLocalDoesNotSubmitWorkflow(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	runner := &fakeLoopRemoteRunner{}
	cmd := &LoopCmd{
		Local:        true,
		Slug:         "fmt",
		Max:          10,
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     &fakeAIClient{},
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
		prClient:     &fakePullRequestOpener{},
		remoteRunner: runner,
	}
	err := cmd.Run()
	require.NoError(t, err)
	assert.False(t, runner.called, "the remote runner is never consulted with --local")
	assert.Equal(t, "fmt", cmd.resolvedSlug, "the loop runs in-process with --local")
}
