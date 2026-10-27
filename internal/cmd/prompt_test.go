package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/services"
)

// promptSlugProposerFunc adapts a function to the command's slug proposer.
type promptSlugProposerFunc func(prompt string) (string, error)

func (f promptSlugProposerFunc) ProposeSlug(prompt string) (string, error) {
	return f(prompt)
}

// promptWorktreeRecorder records worktree operations and returns canned
// results so worktree mode can be tested without git.
type promptWorktreeRecorder struct {
	checkedOut  bool
	path        string
	createErr   error
	checkErr    error
	removeErr   error
	order       *[]string
	checked     []string
	created     []string
	removed     []string
	createCalls int
}

func (w *promptWorktreeRecorder) CreateWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error) {
	w.created = append(w.created, branch)
	w.createCalls++
	promptEvent(w.order, "create-worktree")
	if w.createErr != nil {
		return nil, w.createErr
	}
	path := w.path
	if path == "" {
		path = "/worktree/" + branch
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "add", path, branch}, Path: path}, nil
}

func (w *promptWorktreeRecorder) BranchCheckedOutInWorktree(branch string, dryRun bool) (*git.WorktreeCommand, bool, error) {
	w.checked = append(w.checked, branch)
	promptEvent(w.order, "check-worktree")
	if w.checkErr != nil {
		return nil, false, w.checkErr
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "list", "--porcelain"}}, w.checkedOut, nil
}

func (w *promptWorktreeRecorder) RemoveWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error) {
	w.removed = append(w.removed, branch)
	promptEvent(w.order, "remove-worktree")
	if w.removeErr != nil {
		return nil, w.removeErr
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "remove", branch}}, nil
}

// promptWorkspaceRecorder records directory changes so worktree cleanup can be
// asserted without moving the test process.
type promptWorkspaceRecorder struct {
	dirs  []string
	order *[]string
	err   error
}

func (w *promptWorkspaceRecorder) ChangeDirectory(path string) error {
	w.dirs = append(w.dirs, path)
	promptEvent(w.order, "change-directory")
	return w.err
}

// promptServicesRecorder records the before, start, stop, and log-cleanup calls
// made around the pass and returns canned failures.
type promptServicesRecorder struct {
	order           *[]string
	runBeforeErr    error
	startErr        error
	manager         *services.Manager
	runBeforeCalls  int
	startCalls      int
	stopCalls       int
	removeLogsCalls int
}

func (s *promptServicesRecorder) RunBeforeCommands(cfg *config.RalphConfig) error {
	s.runBeforeCalls++
	promptEvent(s.order, "before")
	return s.runBeforeErr
}

func (s *promptServicesRecorder) Start(cfg *config.RalphConfig) (*services.Manager, error) {
	s.startCalls++
	promptEvent(s.order, "start-services")
	if s.startErr != nil {
		return nil, s.startErr
	}
	if s.manager != nil {
		return s.manager, nil
	}
	return &services.Manager{}, nil
}

func (s *promptServicesRecorder) Stop(svc *services.Manager) {
	s.stopCalls++
	promptEvent(s.order, "stop-services")
}

func (s *promptServicesRecorder) RemoveLogs(cfg *config.RalphConfig) {
	s.removeLogsCalls++
	promptEvent(s.order, "remove-logs")
}

// promptAIRecorder records the prompt pass and the supporting service-fix and
// conflict-resolution passes.
type promptAIRecorder struct {
	order        *[]string
	passFunc     func(string) error
	fixFunc      func(*config.RalphConfig, error) error
	resolveFunc  func(string, string) error
	prompts      []string
	passCalls    int
	fixCalls     int
	resolveCalls int
}

func (a *promptAIRecorder) RunPrompt(prompt string) error {
	a.passCalls++
	a.prompts = append(a.prompts, prompt)
	promptEvent(a.order, "pass")
	if a.passFunc != nil {
		return a.passFunc(prompt)
	}
	return nil
}

func (a *promptAIRecorder) FixServiceStartup(cfg *config.RalphConfig, err error) error {
	a.fixCalls++
	promptEvent(a.order, "service-fix")
	if a.fixFunc != nil {
		return a.fixFunc(cfg, err)
	}
	return nil
}

func (a *promptAIRecorder) ResolveMergeConflicts(baseBranch, promptBranch string) error {
	a.resolveCalls++
	promptEvent(a.order, "resolve-conflicts")
	if a.resolveFunc != nil {
		return a.resolveFunc(baseBranch, promptBranch)
	}
	return nil
}

// promptOutputRecorder captures the warnings emitted during base
// synchronization.
type promptOutputRecorder struct {
	warnings []string
}

func (o *promptOutputRecorder) Warnf(format string, a ...any) {
	o.warnings = append(o.warnings, fmt.Sprintf(format, a...))
}

// promptPRRecorder records the pull request summary input and the branch it
// opens from, and returns a canned failure.
type promptPRRecorder struct {
	order   *[]string
	err     error
	prompts []string
	slugs   []string
	heads   []string
	calls   int
}

func (p *promptPRRecorder) CreatePromptPR(prompt, slug, head string) error {
	p.calls++
	p.prompts = append(p.prompts, prompt)
	p.slugs = append(p.slugs, slug)
	p.heads = append(p.heads, head)
	promptEvent(p.order, "pull-request")
	return p.err
}

// promptEvent appends an event to an optional order log shared by the mock
// clients.
func promptEvent(order *[]string, event string) {
	if order != nil {
		*order = append(*order, event)
	}
}

// preparePromptRun gives a command the mock setup clients its flow needs, so
// tests that only care about an earlier step do not start real services or run
// OpenCode.
func preparePromptRun(cmd *PromptCmd) {
	if cmd.services == nil {
		cmd.services = &promptServicesRecorder{}
	}
	if cmd.ai == nil {
		cmd.ai = &promptAIRecorder{}
	}
	if cmd.output == nil {
		cmd.output = &promptOutputRecorder{}
	}
	if cmd.remote == nil {
		cmd.remote = &promptRemoteRecorder{}
	}
	if cmd.notify == nil {
		cmd.notify = &promptNotifyRecorder{}
	}
}

// TestReadPromptSources covers every prompt input source and the conflicting
// and empty input failures.
func TestReadPromptSources(t *testing.T) {
	dir := t.TempDir()
	filePrompt := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(filePrompt, []byte("from the file\n"), 0644))
	emptyFile := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(emptyFile, []byte("  \n\t"), 0644))

	tests := []struct {
		name     string
		text     []string
		file     string
		stdin    string
		terminal bool
		want     string
		wantErr  string
	}{
		{
			name: "positional text is joined with single spaces",
			text: []string{"update", "the", "readme"},
			want: "update the readme",
		},
		{
			name:     "positional text ignores standard input",
			text:     []string{"use this"},
			stdin:    "not this",
			terminal: true,
			want:     "use this",
		},
		{
			name: "file contents become the prompt",
			file: filePrompt,
			want: "from the file\n",
		},
		{
			name:  "file does not read standard input",
			file:  filePrompt,
			stdin: "not this",
			want:  "from the file\n",
		},
		{
			name:  "piped standard input becomes the prompt",
			stdin: "from standard input",
			want:  "from standard input",
		},
		{
			name:     "explicit standard input is read from a terminal",
			file:     "-",
			stdin:    "explicit standard input",
			terminal: true,
			want:     "explicit standard input",
		},
		{
			name:    "positional text with a file fails",
			text:    []string{"text"},
			file:    filePrompt,
			wantErr: "cannot use positional prompt text with --file",
		},
		{
			name:     "missing prompt fails on a terminal",
			terminal: true,
			wantErr:  "prompt is required",
		},
		{
			name:    "whitespace-only positional text fails",
			text:    []string{" ", "\t"},
			wantErr: "prompt is empty",
		},
		{
			name:    "whitespace-only file fails",
			file:    emptyFile,
			wantErr: "prompt is empty",
		},
		{
			name:    "whitespace-only standard input fails",
			stdin:   " \n\t",
			wantErr: "prompt is empty",
		},
		{
			name:    "missing file fails",
			file:    filepath.Join(dir, "missing.txt"),
			wantErr: "failed to read prompt file",
		},
		{
			name:    "directory path is not a readable prompt file",
			file:    dir,
			wantErr: "failed to read prompt file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadPrompt(tt.text, tt.file, strings.NewReader(tt.stdin), tt.terminal)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// failingReader reports a read failure so the standard input error path is
// covered.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestReadPromptStandardInputError(t *testing.T) {
	_, err := ReadPrompt(nil, "-", failingReader{}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read prompt from standard input")
}

// TestPromptCmdParsing covers the command option surface and rejects the
// project and item options.
func TestPromptCmdParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantPrompt    []string
		wantFile      string
		wantMode      string
		wantBase      string
		wantModel     string
		wantVariant   string
		wantAgent     string
		wantContext   string
		wantNamespace string
		wantNoSvc     bool
		wantNoFollow  bool
		wantNoNotify  bool
		wantDebug     string
		wantVerbose   bool
		wantErr       string
	}{
		{
			name:       "positional tokens are kept in order",
			args:       []string{"prompt", "update", "the", "readme"},
			wantPrompt: []string{"update", "the", "readme"},
		},
		{
			name:     "--file parses",
			args:     []string{"prompt", "--file", "prompt.txt"},
			wantFile: "prompt.txt",
		},
		{
			name:          "all options parse",
			args:          []string{"prompt", "text", "--mode", "remote", "--base", "release", "--model", "gpt-4", "--variant", "high", "--agent", "build", "--context", "prod", "--namespace", "argo", "--no-services", "--no-follow", "--no-notify", "--debug", "feature", "--verbose"},
			wantPrompt:    []string{"text"},
			wantMode:      "remote",
			wantBase:      "release",
			wantModel:     "gpt-4",
			wantVariant:   "high",
			wantAgent:     "build",
			wantContext:   "prod",
			wantNamespace: "argo",
			wantNoSvc:     true,
			wantNoFollow:  true,
			wantNoNotify:  true,
			wantDebug:     "feature",
			wantVerbose:   true,
		},
		{
			name:       "--base short form parses",
			args:       []string{"prompt", "text", "-B", "main"},
			wantPrompt: []string{"text"},
			wantBase:   "main",
		},
		{
			name:          "--namespace short form parses",
			args:          []string{"prompt", "text", "-n", "staging"},
			wantPrompt:    []string{"text"},
			wantNamespace: "staging",
		},
		{
			name:       "leading dash prompt is accepted after the separator",
			args:       []string{"prompt", "--", "--version"},
			wantPrompt: []string{"--version"},
		},
		{
			name:    "positional text with a file is rejected at parse time",
			args:    []string{"prompt", "text", "--file", "prompt.txt"},
			wantErr: "cannot use positional prompt text with --file",
		},
		{
			name:    "--items is rejected",
			args:    []string{"prompt", "text", "--items", "."},
			wantErr: "unknown flag --items",
		},
		{
			name:    "--extra is rejected",
			args:    []string{"prompt", "text", "--extra", "3"},
			wantErr: "unknown flag --extra",
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
			assert.Equal(t, tt.wantPrompt, cmd.Prompt.Prompt)
			assert.Equal(t, tt.wantFile, cmd.Prompt.File)
			assert.Equal(t, tt.wantMode, cmd.Prompt.Mode)
			assert.Equal(t, tt.wantBase, cmd.Prompt.Base)
			assert.Equal(t, tt.wantModel, cmd.Prompt.Model)
			assert.Equal(t, tt.wantVariant, cmd.Prompt.Variant)
			assert.Equal(t, tt.wantAgent, cmd.Prompt.Agent)
			assert.Equal(t, tt.wantContext, cmd.Prompt.Context)
			assert.Equal(t, tt.wantNamespace, cmd.Prompt.Namespace)
			assert.Equal(t, tt.wantNoSvc, cmd.Prompt.NoServices)
			assert.Equal(t, tt.wantNoFollow, cmd.Prompt.NoFollow)
			assert.Equal(t, tt.wantNoNotify, cmd.Prompt.NoNotify)
			assert.Equal(t, tt.wantDebug, cmd.Prompt.Debug)
			assert.Equal(t, tt.wantVerbose, cmd.Prompt.Verbose)
		})
	}
}

func TestPromptCmdHelpListsOptions(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"prompt", "--help"})
	for _, option := range []string{
		"--mode", "--base", "-B", "--model", "--variant", "--agent",
		"--context", "--namespace", "-n", "--no-services", "--no-follow",
		"--no-notify", "--debug", "--verbose",
	} {
		assert.Contains(t, output, option)
	}
	assert.NotContains(t, output, "--items")
	assert.NotContains(t, output, "--extra")
}

func TestPromptCmdListedInTopLevelHelp(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"--help"})
	assert.Contains(t, output, "prompt")
	assert.Contains(t, output, "Run one prompt as a single OpenCode pass")
}

// TestPromptCmdDebugRejection covers --debug for every resolved mode level.
func TestPromptCmdDebugRejection(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		config  func() *config.RalphConfig
		wantErr string
	}{
		{
			name:    "local flag rejects debug",
			mode:    config.ModeLocal,
			config:  func() *config.RalphConfig { return config.Any() },
			wantErr: "--debug flag is not applicable with --mode local",
		},
		{
			name:    "worktree flag rejects debug",
			mode:    config.ModeWorktree,
			config:  func() *config.RalphConfig { return config.Any() },
			wantErr: "--debug flag is not applicable with --mode worktree",
		},
		{
			name:    "local default rejects debug",
			config:  func() *config.RalphConfig { return config.Any() },
			wantErr: "--debug flag is not applicable with --mode local",
		},
		{
			name:    "config worktree rejects debug",
			config:  func() *config.RalphConfig { return config.WithMode(config.ModeWorktree) },
			wantErr: "--debug flag is not applicable with --mode worktree",
		},
		{
			name:   "remote flag accepts debug",
			mode:   config.ModeRemote,
			config: func() *config.RalphConfig { return config.Any() },
		},
		{
			name:   "config remote accepts debug",
			config: func() *config.RalphConfig { return config.WithMode(config.ModeRemote) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &PromptCmd{
				Prompt: []string{"do work"},
				Mode:   tt.mode,
				Debug:  "feature",
				config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
					return tt.config(), nil
				}},
				git:          &git.MockClient{CurrentBranchFunc: func() (string, error) { return "main", nil }},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "do-work", nil }),
				worktree:     &promptWorktreeRecorder{},
				workspace:    &promptWorkspaceRecorder{},
			}
			preparePromptRun(cmd)
			err := cmd.Run()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestPromptCmdInputErrorsPrecedeModeWork asserts a bad prompt is rejected
// before the config is loaded for mode resolution.
func TestPromptCmdInputErrorsPrecedeModeWork(t *testing.T) {
	loadErr := errors.New("config should not be loaded")
	cmd := &PromptCmd{
		Prompt:          []string{" "},
		stdinIsTerminal: func() bool { return true },
		config:          &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return nil, loadErr }},
	}
	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is empty")
	assert.NotContains(t, err.Error(), loadErr.Error())
}

// TestPromptCmdModeResolution covers every mode level and an invalid mode at
// both the flag and config levels. An invalid mode fails before the current
// branch is read.
func TestPromptCmdModeResolution(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		config  *config.RalphConfig
		want    string
		wantErr string
	}{
		{
			name:   "flag local overrides config",
			mode:   config.ModeLocal,
			config: config.WithMode(config.ModeRemote),
			want:   config.ModeLocal,
		},
		{
			name:   "flag worktree overrides config",
			mode:   config.ModeWorktree,
			config: config.WithMode(config.ModeLocal),
			want:   config.ModeWorktree,
		},
		{
			name:   "flag remote overrides config",
			mode:   config.ModeRemote,
			config: config.WithMode(config.ModeWorktree),
			want:   config.ModeRemote,
		},
		{
			name:   "config worktree without a flag",
			config: config.WithMode(config.ModeWorktree),
			want:   config.ModeWorktree,
		},
		{
			name:   "config remote without a flag",
			config: config.WithMode(config.ModeRemote),
			want:   config.ModeRemote,
		},
		{
			name:   "local is the default",
			config: &config.RalphConfig{DefaultBranch: "main"},
			want:   config.ModeLocal,
		},
		{
			name:    "invalid flag is rejected",
			mode:    "sandbox",
			config:  config.WithMode(config.ModeRemote),
			wantErr: "invalid mode: sandbox (expected local, worktree, or remote)",
		},
		{
			name:    "invalid config is rejected",
			config:  config.WithMode("sandbox"),
			wantErr: "invalid mode: sandbox (expected local, worktree, or remote)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchReads := 0
			slugProposals := 0
			cmd := &PromptCmd{
				Prompt: []string{"do work"},
				Mode:   tt.mode,
				config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
					return tt.config, nil
				}},
				git: &git.MockClient{CurrentBranchFunc: func() (string, error) {
					branchReads++
					return "main", nil
				}},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) {
					slugProposals++
					return "do-work", nil
				}),
				worktree:  &promptWorktreeRecorder{},
				workspace: &promptWorkspaceRecorder{},
			}

			preparePromptRun(cmd)
			err := cmd.Run()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, cmd.resolved)
				assert.Zero(t, branchReads, "an invalid mode must not start branch work")
				assert.Zero(t, slugProposals, "an invalid mode must not propose a slug")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cmd.resolved)
			assert.Equal(t, tt.want, cmd.resolved.Mode)
			assert.Equal(t, 1, branchReads)
			assert.Equal(t, 1, slugProposals)
		})
	}
}

// TestPromptCmdBaseResolution covers every base branch level, including the
// current branch already being the prompt branch.
func TestPromptCmdBaseResolution(t *testing.T) {
	tests := []struct {
		name          string
		base          string
		current       string
		promptBranch  string
		defaultBranch string
		want          string
	}{
		{
			name:          "base flag wins",
			base:          "release",
			current:       "main",
			promptBranch:  "feature",
			defaultBranch: "develop",
			want:          "release",
		},
		{
			name:          "current branch differs from the prompt branch",
			current:       "main",
			promptBranch:  "feature",
			defaultBranch: "develop",
			want:          "main",
		},
		{
			name:          "config default when already on the prompt branch",
			current:       "feature",
			promptBranch:  "feature",
			defaultBranch: "develop",
			want:          "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Any()
			cfg.DefaultBranch = tt.defaultBranch
			cmd := &PromptCmd{
				Prompt: []string{"do work"},
				Base:   tt.base,
				config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
					return cfg, nil
				}},
				git: &git.MockClient{CurrentBranchFunc: func() (string, error) {
					return tt.current, nil
				}},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) {
					return tt.promptBranch, nil
				}),
			}

			preparePromptRun(cmd)
			require.NoError(t, cmd.Run())
			require.NotNil(t, cmd.resolved)
			assert.Equal(t, tt.want, cmd.resolved.BaseBranch)
			assert.Equal(t, tt.current, cmd.resolved.CurrentBranch)
			assert.Equal(t, tt.promptBranch, cmd.resolved.Slug)
			assert.Equal(t, tt.promptBranch, cmd.resolved.PromptBranch)
			assert.Equal(t, tt.want, cfg.Base)
			assert.Equal(t, tt.current, cfg.StartingBranch)
		})
	}
}

// TestPromptCmdCurrentBranchError covers a failure to read the branch the
// prompt starts from.
func TestPromptCmdCurrentBranchError(t *testing.T) {
	cmd := &PromptCmd{
		Prompt: []string{"do work"},
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{CurrentBranchFunc: func() (string, error) {
			return "", errors.New("not a git repository")
		}},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "do-work", nil }),
	}

	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
	assert.Nil(t, cmd.resolved)
}

// TestPromptCmdResolvesModeAndPrompt covers the successful command path with
// standard input injected.
func TestPromptCmdResolvesModeAndPrompt(t *testing.T) {
	cmd := &PromptCmd{
		stdin:           strings.NewReader("piped prompt"),
		stdinIsTerminal: func() bool { return false },
		config:          &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:             &git.MockClient{CurrentBranchFunc: func() (string, error) { return "main", nil }},
		slugProposer:    promptSlugProposerFunc(func(string) (string, error) { return "piped-prompt", nil }),
	}
	preparePromptRun(cmd)
	require.NoError(t, cmd.Run())
	require.NotNil(t, cmd.resolved)
	assert.Equal(t, config.ModeLocal, cmd.resolved.Mode)
}

// TestPromptCmdSlugCleanup covers cleaning a proposed slug like a project
// branch before it becomes the prompt branch name.
func TestPromptCmdSlugCleanup(t *testing.T) {
	tests := []struct {
		name     string
		proposal string
		want     string
	}{
		{
			name:     "spaces become hyphens",
			proposal: "update the readme",
			want:     "update-the-readme",
		},
		{
			name:     "capitals are lowered",
			proposal: "UPPER_CASE",
			want:     "upper-case",
		},
		{
			name:     "special characters are removed",
			proposal: "Fix: Auth/Bug!",
			want:     "fix-authbug",
		},
		{
			name:     "dots, underscores, and surrounding space are cleaned",
			proposal: "  My_Feature.Work  ",
			want:     "my-feature-work",
		},
		{
			name:     "an already clean slug is unchanged",
			proposal: "already-clean",
			want:     "already-clean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var switched string
			cmd := &PromptCmd{
				Prompt:       []string{"do work"},
				config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				git:          &git.MockClient{SwitchToBranchFunc: func(slug string) error { switched = slug; return nil }},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { return tt.proposal, nil }),
			}

			preparePromptRun(cmd)
			require.NoError(t, cmd.Run())
			require.NotNil(t, cmd.resolved)
			assert.Equal(t, tt.want, cmd.resolved.PromptBranch)
			assert.Equal(t, tt.want, switched)
		})
	}
}

// TestPromptCmdFailedSlugProposalStopsBeforeBranchWork asserts a failed
// proposal returns before the current branch is read, the branch is switched,
// or a worktree is created.
func TestPromptCmdFailedSlugProposalStopsBeforeBranchWork(t *testing.T) {
	proposalErr := errors.New("opencode execution failed: quota")
	branchReads := 0
	switchCalls := 0
	worktree := &promptWorktreeRecorder{}
	cmd := &PromptCmd{
		Prompt: []string{"do work"},
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { branchReads++; return "main", nil },
			SwitchToBranchFunc: func(string) error {
				switchCalls++
				return nil
			},
		},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "", proposalErr }),
		worktree:     worktree,
		workspace:    &promptWorkspaceRecorder{},
	}

	err := cmd.Run()
	require.Error(t, err)
	assert.ErrorIs(t, err, proposalErr)
	assert.Nil(t, cmd.resolved)
	assert.Zero(t, branchReads, "a failed proposal must not read the starting branch")
	assert.Zero(t, switchCalls, "a failed proposal must not switch branches")
	assert.Empty(t, worktree.created, "a failed proposal must not create a worktree")
}

// TestPromptCmdLocalBranches covers local mode switching the current checkout
// to the prompt branch.
func TestPromptCmdLocalBranches(t *testing.T) {
	var switched string
	currentReads := 0
	worktree := &promptWorktreeRecorder{}
	cmd := &PromptCmd{
		Prompt: []string{"do work"},
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { currentReads++; return "main", nil },
			SwitchToBranchFunc: func(slug string) error {
				switched = slug
				return nil
			},
		},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		worktree:     worktree,
	}

	preparePromptRun(cmd)
	require.NoError(t, cmd.Run())
	assert.Equal(t, 1, currentReads)
	assert.Equal(t, "my-feature", switched)
	assert.Equal(t, "my-feature", cmd.resolved.PromptBranch)
	assert.Empty(t, worktree.created, "local mode creates no worktree")
}

// TestPromptCmdWorktreeRefusesCheckedOutBranch covers refusing a prompt branch
// already checked out in another worktree before creating one.
func TestPromptCmdWorktreeRefusesCheckedOutBranch(t *testing.T) {
	switchCalls := 0
	worktree := &promptWorktreeRecorder{checkedOut: true}
	cmd := &PromptCmd{
		Prompt: []string{"do work"},
		Mode:   config.ModeWorktree,
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			SwitchToBranchFunc: func(string) error {
				switchCalls++
				return nil
			},
		},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		worktree:     worktree,
		workspace:    &promptWorkspaceRecorder{},
	}

	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch 'my-feature' is already checked out in another worktree")
	assert.Equal(t, []string{"my-feature"}, worktree.checked)
	assert.Zero(t, worktree.createCalls, "no worktree is created for a refused branch")
	assert.Empty(t, worktree.removed)
	assert.Zero(t, switchCalls, "worktree mode does not switch the starting checkout")
}

// TestPromptCmdWorktreeCreatesAndRemoves covers creating the worktree on the
// prompt branch, working inside it, and removing it when the command ends while
// the starting checkout stays on its original branch.
func TestPromptCmdWorktreeCreatesAndRemoves(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	switchCalls := 0
	worktree := &promptWorktreeRecorder{path: "/worktree/my-feature"}
	workspace := &promptWorkspaceRecorder{}
	cmd := &PromptCmd{
		Prompt: []string{"do work"},
		Mode:   config.ModeWorktree,
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			SwitchToBranchFunc: func(string) error {
				switchCalls++
				return nil
			},
		},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		worktree:     worktree,
		workspace:    workspace,
	}

	preparePromptRun(cmd)
	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{"my-feature"}, worktree.checked)
	assert.Equal(t, []string{"my-feature"}, worktree.created)
	assert.Equal(t, []string{"my-feature"}, worktree.removed)
	assert.Equal(t, []string{"/worktree/my-feature", originalDir}, workspace.dirs)
	assert.Zero(t, switchCalls, "the starting checkout stays on its original branch")
	assert.Equal(t, "main", cmd.resolved.CurrentBranch)
}

// TestPromptCmdSetupOrder covers the pre-pass order in local mode: the before
// commands run before the branch switch, then the base branch is synchronized,
// the services start, and the pass runs.
func TestPromptCmdSetupOrder(t *testing.T) {
	var order []string
	servicesClient := &promptServicesRecorder{order: &order}
	aiClient := &promptAIRecorder{order: &order}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     servicesClient,
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			SwitchToBranchFunc: func(string) error {
				promptEvent(&order, "switch-branch")
				return nil
			},
			FetchBranchFunc: func(branch string) error {
				assert.Equal(t, "main", branch)
				promptEvent(&order, "fetch-base")
				return nil
			},
			NeedsMergeFunc: func(string) (bool, error) {
				promptEvent(&order, "needs-merge")
				return false, nil
			},
		},
	}

	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{
		"before",
		"switch-branch",
		"fetch-base",
		"needs-merge",
		"start-services",
		"pass",
		"stop-services",
		"remove-logs",
	}, order)
	assert.Equal(t, []string{"do work"}, aiClient.prompts)
	assert.Equal(t, 1, servicesClient.runBeforeCalls)
	assert.Equal(t, 1, servicesClient.startCalls)
	assert.Equal(t, 1, servicesClient.stopCalls)
}

// TestPromptCmdWorktreeBeforeCommandsRunInside covers the before commands,
// base synchronization, services, and pass all running inside the new
// worktree.
func TestPromptCmdWorktreeBeforeCommandsRunInside(t *testing.T) {
	var order []string
	worktree := &promptWorktreeRecorder{path: "/worktree/my-feature", order: &order}
	workspace := &promptWorkspaceRecorder{order: &order}
	servicesClient := &promptServicesRecorder{order: &order}
	aiClient := &promptAIRecorder{order: &order}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		Mode:         config.ModeWorktree,
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     servicesClient,
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		worktree:     worktree,
		workspace:    workspace,
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			FetchBranchFunc: func(string) error {
				promptEvent(&order, "fetch-base")
				return nil
			},
			NeedsMergeFunc: func(string) (bool, error) {
				promptEvent(&order, "needs-merge")
				return false, nil
			},
		},
	}

	require.NoError(t, cmd.Run())
	before := promptIndex(order, "before")
	changeDir := promptIndex(order, "change-directory")
	fetch := promptIndex(order, "fetch-base")
	start := promptIndex(order, "start-services")
	pass := promptIndex(order, "pass")
	require.NotEqual(t, -1, before)
	assert.Less(t, changeDir, before, "before commands run after entering the worktree")
	assert.Less(t, before, fetch, "the before commands run before synchronization")
	assert.Less(t, fetch, start, "synchronization runs before services start")
	assert.Less(t, start, pass, "services start before the pass")
	assert.Equal(t, []string{"my-feature"}, worktree.removed)
}

// TestPromptCmdRequiredBeforeFailure covers a required before command failing
// and stopping the command before any later setup or pass work.
func TestPromptCmdRequiredBeforeFailure(t *testing.T) {
	beforeErr := errors.New("required before command failed")
	var order []string
	servicesClient := &promptServicesRecorder{order: &order, runBeforeErr: beforeErr}
	aiClient := &promptAIRecorder{order: &order}
	switchCalls := 0
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     servicesClient,
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			SwitchToBranchFunc: func(string) error {
				switchCalls++
				return nil
			},
			FetchBranchFunc: func(string) error {
				promptEvent(&order, "fetch-base")
				return nil
			},
		},
	}

	err := cmd.Run()
	require.ErrorIs(t, err, beforeErr)
	assert.Equal(t, []string{"before"}, order)
	assert.Zero(t, switchCalls, "a failed required before command must not switch branches")
	assert.Zero(t, servicesClient.startCalls, "a failed required before command must not start services")
	assert.Zero(t, aiClient.passCalls, "a failed required before command must not run the pass")
}

// TestPromptCmdBeforeHelper exits non-zero when run as a before command by the
// optional-failure test. The environment variable keeps it inert during the
// normal test run.
func TestPromptCmdBeforeHelper(t *testing.T) {
	if os.Getenv("RALPH_PROMPT_BEFORE_HELPER") != "1" {
		return
	}
	os.Exit(1)
}

// TestPromptCmdOptionalBeforeFailureContinues covers an optional before command
// failing: Ralph warns, continues the setup, and runs the pass.
func TestPromptCmdOptionalBeforeFailureContinues(t *testing.T) {
	t.Setenv("RALPH_PROMPT_BEFORE_HELPER", "1")
	var outBuf strings.Builder
	cfg := config.Any()
	cfg.Before = []config.Before{{
		Name:     "optional",
		Command:  os.Args[0],
		Args:     []string{"-test.run=TestPromptCmdBeforeHelper"},
		Optional: true,
	}}
	aiClient := &promptAIRecorder{}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return cfg, nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     services.NewClient(output.NewClient(&outBuf, &outBuf, false)),
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
		},
	}

	require.NoError(t, cmd.Run())
	assert.Contains(t, outBuf.String(), "Optional before optional failed")
	assert.Equal(t, 1, aiClient.passCalls)
}

// TestPromptCmdServiceFixSuccess covers a service startup failure repaired by
// the service-fix prompt: the pass continues without services.
func TestPromptCmdServiceFixSuccess(t *testing.T) {
	startErr := errors.New("service failed to start")
	var order []string
	servicesClient := &promptServicesRecorder{order: &order, startErr: startErr}
	aiClient := &promptAIRecorder{order: &order}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     servicesClient,
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
		},
	}

	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{"before", "start-services", "service-fix", "pass", "remove-logs"}, order)
	assert.Equal(t, 1, aiClient.fixCalls)
	assert.Equal(t, 1, aiClient.passCalls)
	assert.Zero(t, servicesClient.stopCalls, "a fixed startup failure continues without services")
}

// TestPromptCmdServiceFixFailure covers a service startup failure whose fix
// prompt also fails: the command fails and the pass does not run.
func TestPromptCmdServiceFixFailure(t *testing.T) {
	startErr := errors.New("service failed to start")
	fixErr := errors.New("service fix failed")
	var order []string
	servicesClient := &promptServicesRecorder{order: &order, startErr: startErr}
	aiClient := &promptAIRecorder{order: &order, fixFunc: func(*config.RalphConfig, error) error { return fixErr }}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     servicesClient,
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
		},
	}

	err := cmd.Run()
	require.ErrorIs(t, err, fixErr)
	assert.Equal(t, []string{"before", "start-services", "service-fix", "remove-logs"}, order)
	assert.Zero(t, aiClient.passCalls, "a failed service fix must not run the pass")
	assert.Zero(t, servicesClient.stopCalls)
}

// TestPromptCmdNoServices covers --no-services skipping service startup and the
// fix prompt while still running the before commands and the pass.
func TestPromptCmdNoServices(t *testing.T) {
	startErr := errors.New("service failed to start")
	var order []string
	servicesClient := &promptServicesRecorder{order: &order, startErr: startErr}
	aiClient := &promptAIRecorder{order: &order, fixFunc: func(*config.RalphConfig, error) error {
		return errors.New("the fix prompt must not run")
	}}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		NoServices:   true,
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		services:     servicesClient,
		ai:           aiClient,
		output:       &promptOutputRecorder{},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
		},
	}

	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{"before", "pass", "remove-logs"}, order)
	assert.Equal(t, 1, servicesClient.runBeforeCalls)
	assert.Zero(t, servicesClient.startCalls, "--no-services must not start services")
	assert.Zero(t, aiClient.fixCalls, "--no-services must not run the service-fix prompt")
	assert.Equal(t, 1, aiClient.passCalls)
}

// TestPromptCmdExecutionContextCarriesAISelection covers the command flags
// reaching the execution context used by the pass, the slug, and the git
// clients. Custom instructions are never placed on the context, so the direct
// prompt runs as written.
func TestPromptCmdExecutionContextCarriesAISelection(t *testing.T) {
	cmd := &PromptCmd{
		Model:   "gpt-4",
		Variant: "high",
		Agent:   "build",
		Verbose: true,
	}

	ctx := cmd.executionContext()
	assert.Equal(t, "gpt-4", ctx.Model())
	assert.Equal(t, "high", ctx.Variant())
	assert.Equal(t, "build", ctx.Agent())
	assert.True(t, ctx.IsVerbose())
	assert.Empty(t, ctx.InstructionsMD(), "custom instructions must not apply to a direct prompt")
	assert.Same(t, ctx, cmd.executionContext(), "the command resolves its execution context once")
}

// TestPromptAIClientRunPrompt covers the direct prompt pass: one non-interactive
// pass with the prompt as written, the configured model, variant, and agent, no
// custom instructions, and a failure that is returned without a retry.
func TestPromptAIClientRunPrompt(t *testing.T) {
	newAdapter := func(t *testing.T, mock *opencode.MockOC) *promptAIClientAdapter {
		ctx := execcontext.NewContext()
		ctx.SetModel("gpt-4")
		ctx.SetVariant("high")
		ctx.SetAgent("build")
		return &promptAIClientAdapter{ctx: ctx, oc: mock}
	}

	t.Run("runs one pass with the configured model, variant, and agent", func(t *testing.T) {
		var calls int
		var gotModel, gotVariant, gotAgent, gotPrompt string
		mock := &opencode.MockOC{RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
			calls++
			gotModel, gotVariant, gotAgent, gotPrompt = model, variant, agent, prompt
			return nil
		}}
		prompt := "Add the endpoint.\n\nRun the tests before committing.\n"
		require.NoError(t, newAdapter(t, mock).RunPrompt(prompt))
		assert.Equal(t, 1, calls, "the prompt runs as one pass")
		assert.Equal(t, "gpt-4", gotModel)
		assert.Equal(t, "high", gotVariant)
		assert.Equal(t, "build", gotAgent)
		assert.Equal(t, prompt, gotPrompt, "the prompt is sent exactly as written")
	})

	t.Run("leaves tool access to the agent", func(t *testing.T) {
		var gotPrompt string
		mock := &opencode.MockOC{RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			gotPrompt = prompt
			return nil
		}}
		prompt := "Use git and shell commands to commit the change yourself."
		require.NoError(t, newAdapter(t, mock).RunPrompt(prompt))
		assert.Equal(t, prompt, gotPrompt, "the prompt is unchanged, so no tool restrictions are added")
	})

	t.Run("custom instructions do not apply", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "instructions.md"), []byte("Custom instructions: always add a changelog"), 0644))
		t.Chdir(dir)

		var gotPrompt string
		mock := &opencode.MockOC{RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			gotPrompt = prompt
			return nil
		}}
		prompt := "Fix the typo in the readme."
		require.NoError(t, newAdapter(t, mock).RunPrompt(prompt))
		assert.Equal(t, prompt, gotPrompt)
		assert.NotContains(t, gotPrompt, "Custom instructions", "a direct prompt must not receive the repository's custom instructions")
	})

	t.Run("each invocation starts a fresh pass", func(t *testing.T) {
		var prompts []string
		mock := &opencode.MockOC{RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			prompts = append(prompts, prompt)
			return nil
		}}
		adapter := newAdapter(t, mock)
		require.NoError(t, adapter.RunPrompt("first prompt"))
		require.NoError(t, adapter.RunPrompt("second prompt"))
		assert.Equal(t, []string{"first prompt", "second prompt"}, prompts, "every invocation is its own pass")
	})

	t.Run("a failing pass fails without retrying", func(t *testing.T) {
		passErr := errors.New("opencode execution failed: exit status 1")
		calls := 0
		mock := &opencode.MockOC{RunAgentFunc: func(_ context.Context, _, _, _, _ string) error {
			calls++
			return passErr
		}}
		err := newAdapter(t, mock).RunPrompt("do work")
		require.ErrorIs(t, err, passErr)
		assert.Equal(t, 1, calls, "a failed pass is not retried")
	})
}

// TestPromptAIClientResolveMergeConflictsUsesConfiguredAgent covers the adapter
// routing conflict resolution to the configured agent, because it writes
// repository code.
func TestPromptAIClientResolveMergeConflictsUsesConfiguredAgent(t *testing.T) {
	ctx := execcontext.NewContext()
	ctx.SetAgent("build")

	var capturedAgent string
	mock := &opencode.MockOC{RunAgentFunc: func(_ context.Context, _, _, agent, _ string) error {
		capturedAgent = agent
		return nil
	}}
	adapter := &promptAIClientAdapter{ctx: ctx, oc: mock, agent: NewAgentClient(ctx, mock)}

	require.NoError(t, adapter.ResolveMergeConflicts("main", "feature"))
	assert.Equal(t, "build", capturedAgent)
}

// TestPromptCmdRunsOnePassWithoutReportOrBlockContract covers the command
// running exactly one pass and reading no report or blocked file. Both are
// directories, so a read would fail the command.
func TestPromptCmdRunsOnePassWithoutReportOrBlockContract(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "report.md"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "blocked.md"), 0755))
	t.Chdir(dir)

	aiClient := &promptAIRecorder{}
	cmd := &PromptCmd{
		Prompt:       []string{"do", "work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		ai:           aiClient,
		git:          &git.MockClient{},
	}
	preparePromptRun(cmd)

	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{"do work"}, aiClient.prompts)
	assert.Equal(t, 1, aiClient.passCalls, "the prompt runs exactly once")
	assert.Zero(t, aiClient.fixCalls)
	assert.Zero(t, aiClient.resolveCalls)
}

// TestPromptCmdPassFailureFailsCommand covers a non-zero OpenCode exit failing
// the command after a single pass.
func TestPromptCmdPassFailureFailsCommand(t *testing.T) {
	passErr := errors.New("opencode execution failed: exit status 1")
	aiClient := &promptAIRecorder{passFunc: func(string) error { return passErr }}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		ai:           aiClient,
		git:          &git.MockClient{},
	}
	preparePromptRun(cmd)

	err := cmd.Run()
	require.ErrorIs(t, err, passErr)
	assert.Equal(t, 1, aiClient.passCalls, "a failed pass is not retried")
}

// TestPromptCmdTipComparison covers the branch tip recorded after base
// synchronization and compared after a successful pass. An unchanged tip skips
// the second synchronization and the push, leaving uncommitted changes in a
// local checkout and discarding them with the worktree. An advanced tip
// synchronizes the base branch again and pushes the prompt branch. A failed
// pass makes no push.
func TestPromptCmdTipComparison(t *testing.T) {
	passErr := errors.New("opencode execution failed: exit status 1")

	tests := []struct {
		name          string
		mode          string
		advance       bool
		passErr       error
		wantFetches   int
		wantMerges    int
		wantMergedRef string
		wantPush      bool
		wantRemoved   bool
	}{
		{
			name:        "local unchanged tip skips the second sync and the push",
			mode:        config.ModeLocal,
			wantFetches: 1,
		},
		{
			name:          "local advanced tip synchronizes again and pushes",
			mode:          config.ModeLocal,
			advance:       true,
			wantFetches:   2,
			wantMerges:    1,
			wantMergedRef: "main",
			wantPush:      true,
		},
		{
			name:        "worktree unchanged tip discards uncommitted changes with the worktree",
			mode:        config.ModeWorktree,
			wantFetches: 1,
			wantRemoved: true,
		},
		{
			name:          "worktree advanced tip synchronizes again and pushes",
			mode:          config.ModeWorktree,
			advance:       true,
			wantFetches:   2,
			wantMerges:    1,
			wantMergedRef: "origin/main",
			wantPush:      true,
			wantRemoved:   true,
		},
		{
			name:        "local failed pass skips the second sync and the push",
			mode:        config.ModeLocal,
			passErr:     passErr,
			wantFetches: 1,
		},
		{
			name:        "worktree failed pass skips the second sync and the push",
			mode:        config.ModeWorktree,
			passErr:     passErr,
			wantFetches: 1,
			wantRemoved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tip := "tip-before"
			fetches := 0
			merges := 0
			pushes := 0
			needsMergeCalls := 0
			mergedRef := ""
			var order []string

			gitClient := &git.MockClient{
				CurrentBranchFunc: func() (string, error) { return "main", nil },
				HeadCommitFunc:    func() (string, error) { return tip, nil },
				FetchBranchFunc: func(branch string) error {
					assert.Equal(t, "main", branch)
					fetches++
					promptEvent(&order, "fetch-base")
					return nil
				},
				NeedsMergeFunc: func(string) (bool, error) {
					needsMergeCalls++
					return needsMergeCalls == 2, nil
				},
				MergeFunc: func(branch string) error {
					merges++
					mergedRef = branch
					promptEvent(&order, "merge-base")
					return nil
				},
				PushFunc: func() error {
					pushes++
					promptEvent(&order, "push")
					return nil
				},
				CheckoutBranchFunc: func(name string) error {
					promptEvent(&order, "checkout-"+name)
					return nil
				},
			}
			aiClient := &promptAIRecorder{
				order: &order,
				passFunc: func(string) error {
					if tt.advance {
						tip = "tip-after"
					}
					return tt.passErr
				},
			}
			prClient := &promptPRRecorder{order: &order}
			worktree := &promptWorktreeRecorder{path: "/worktree/my-feature"}
			workspace := &promptWorkspaceRecorder{}
			cmd := &PromptCmd{
				Prompt:       []string{"do work"},
				Mode:         tt.mode,
				config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
				git:          gitClient,
				worktree:     worktree,
				workspace:    workspace,
				services:     &promptServicesRecorder{order: &order},
				ai:           aiClient,
				pr:           prClient,
				output:       &promptOutputRecorder{},
			}

			err := cmd.Run()
			if tt.passErr != nil {
				require.ErrorIs(t, err, tt.passErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantFetches, fetches)
			assert.Equal(t, tt.wantMerges, merges)
			assert.Equal(t, tt.wantMergedRef, mergedRef)
			assert.Equal(t, tt.wantPush, pushes > 0)
			if tt.wantPush {
				assert.Less(t, promptIndex(order, "pass"), promptIndex(order, "push"), "the push follows the pass")
				assert.Less(t, promptIndex(order, "push"), promptIndex(order, "pull-request"), "the pull request follows the push")
			} else {
				assert.Zero(t, prClient.calls, "no pull request opens without a push")
			}
			if tt.advance {
				assert.Less(t, promptIndex(order, "pass"), promptLastIndex(order, "fetch-base"), "the second synchronization follows the pass")
			}
			if tt.wantRemoved {
				assert.Equal(t, []string{"my-feature"}, worktree.removed)
			} else {
				assert.Empty(t, worktree.created, "local mode does not create a worktree")
				assert.Empty(t, worktree.removed, "local mode keeps uncommitted changes in the checkout")
			}
		})
	}
}

// TestPromptCmdRecordsTipAfterFirstSync covers recording the tip after the
// first synchronization, so a merge performed by that synchronization is not
// mistaken for a prompt commit when the pass leaves the tip unchanged.
func TestPromptCmdRecordsTipAfterFirstSync(t *testing.T) {
	tip := "tip-before"
	merges := 0
	pushes := 0
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			HeadCommitFunc:    func() (string, error) { return tip, nil },
			NeedsMergeFunc:    func(string) (bool, error) { return merges == 0, nil },
			MergeFunc: func(string) error {
				merges++
				tip = "tip-after-merge"
				return nil
			},
			PushFunc: func() error {
				pushes++
				return nil
			},
		},
		output: &promptOutputRecorder{},
	}
	preparePromptRun(cmd)

	require.NoError(t, cmd.Run())
	assert.Equal(t, 1, merges, "the first synchronization merges the base branch")
	assert.Zero(t, pushes, "a merge from the first synchronization is not a prompt commit")
}

// TestPromptCmdPushFailureFails covers a push failure after a prompt commit.
func TestPromptCmdPushFailureFails(t *testing.T) {
	pushErr := errors.New("push failed")
	tip := "tip-before"
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			HeadCommitFunc:    func() (string, error) { return tip, nil },
			PushFunc:          func() error { return pushErr },
		},
		ai:     &promptAIRecorder{passFunc: func(string) error { tip = "tip-after"; return nil }},
		output: &promptOutputRecorder{},
	}
	preparePromptRun(cmd)

	err := cmd.Run()
	require.ErrorIs(t, err, pushErr)
}

// TestPromptCmdSecondSyncFailureSkipsPush covers a failed second
// synchronization: the command fails before pushing anything.
func TestPromptCmdSecondSyncFailureSkipsPush(t *testing.T) {
	syncErr := errors.New("cannot compare branches")
	pushCalled := false
	needsMergeCalls := 0
	tip := "tip-before"
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			HeadCommitFunc:    func() (string, error) { return tip, nil },
			NeedsMergeFunc: func(string) (bool, error) {
				needsMergeCalls++
				if needsMergeCalls == 2 {
					return false, syncErr
				}
				return false, nil
			},
			PushFunc: func() error {
				pushCalled = true
				return nil
			},
		},
		ai:     &promptAIRecorder{passFunc: func(string) error { tip = "tip-after"; return nil }},
		output: &promptOutputRecorder{},
	}
	preparePromptRun(cmd)

	err := cmd.Run()
	require.ErrorIs(t, err, syncErr)
	assert.False(t, pushCalled, "a failed second synchronization pushes nothing")
}

// TestPromptCmdTipReadFailure fails the command when the branch tip cannot be
// read after the first synchronization or after the pass.
func TestPromptCmdTipReadFailure(t *testing.T) {
	readErr := errors.New("cannot read branch tip")
	reads := 0
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			HeadCommitFunc: func() (string, error) {
				reads++
				if reads == 2 {
					return "", readErr
				}
				return "tip", nil
			},
		},
		output: &promptOutputRecorder{},
	}
	preparePromptRun(cmd)

	err := cmd.Run()
	require.ErrorIs(t, err, readErr)
	assert.Equal(t, 2, reads)
}

// TestPromptCmdPullRequestAfterPush covers the pull request opened after a
// successful push: the summary receives the prompt text, the pull request uses
// the prompt branch, a failure after the push leaves the pushed branch, and a
// local success restores the starting branch while worktree mode leaves the
// starting checkout untouched.
func TestPromptCmdPullRequestAfterPush(t *testing.T) {
	prErr := errors.New("pull request summary failed")
	tests := []struct {
		name         string
		mode         string
		prErr        error
		wantCheckout []string
		wantRemoved  bool
	}{
		{
			name:         "local success restores the starting branch",
			mode:         config.ModeLocal,
			wantCheckout: []string{"main"},
		},
		{
			name:        "worktree success leaves the starting checkout untouched",
			mode:        config.ModeWorktree,
			wantRemoved: true,
		},
		{
			name:  "local failure after the push leaves the pushed branch",
			mode:  config.ModeLocal,
			prErr: prErr,
		},
		{
			name:        "worktree failure after the push leaves the pushed branch",
			mode:        config.ModeWorktree,
			prErr:       prErr,
			wantRemoved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tip := "tip-before"
			pushes := 0
			var checkouts []string
			var order []string
			prClient := &promptPRRecorder{order: &order, err: tt.prErr}
			worktree := &promptWorktreeRecorder{path: "/worktree/my-feature"}
			cmd := &PromptCmd{
				Prompt:       []string{"do work"},
				Mode:         tt.mode,
				config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
				services:     &promptServicesRecorder{order: &order},
				ai: &promptAIRecorder{
					order:    &order,
					passFunc: func(string) error { tip = "tip-after"; return nil },
				},
				pr:        prClient,
				output:    &promptOutputRecorder{},
				worktree:  worktree,
				workspace: &promptWorkspaceRecorder{},
				git: &git.MockClient{
					CurrentBranchFunc: func() (string, error) { return "main", nil },
					HeadCommitFunc:    func() (string, error) { return tip, nil },
					PushFunc: func() error {
						pushes++
						promptEvent(&order, "push")
						return nil
					},
					CheckoutBranchFunc: func(name string) error {
						checkouts = append(checkouts, name)
						promptEvent(&order, "checkout-"+name)
						return nil
					},
				},
			}

			err := cmd.Run()
			if tt.prErr != nil {
				require.ErrorIs(t, err, tt.prErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, 1, pushes, "the branch is pushed before the pull request step")
			require.Equal(t, 1, prClient.calls, "the advanced tip opens one pull request")
			assert.Equal(t, []string{"do work"}, prClient.prompts, "the summary receives the prompt text")
			assert.Equal(t, []string{"my-feature"}, prClient.slugs, "the summary title falls back to the prompt slug")
			assert.Equal(t, []string{"my-feature"}, prClient.heads, "the pull request opens from the prompt branch")
			assert.Less(t, promptIndex(order, "push"), promptIndex(order, "pull-request"), "the pull request follows the push")
			assert.Equal(t, tt.wantCheckout, checkouts)
			if len(tt.wantCheckout) > 0 {
				assert.Less(t, promptIndex(order, "pull-request"), promptIndex(order, "checkout-main"), "the starting branch is restored after the pull request")
			}
			if tt.wantRemoved {
				assert.Equal(t, []string{"my-feature"}, worktree.removed)
			} else {
				assert.Empty(t, worktree.created)
			}
		})
	}
}

// promptRemoteRecorder records the submissions and log operations made for a
// remote prompt and returns canned failures.
type promptRemoteRecorder struct {
	submissions   []promptRemoteSubmission
	submitErr     error
	followErr     error
	hints         []string
	followed      []string
	submitCalls   int
	hintCalls     int
	followLogsCnt int
}

func (r *promptRemoteRecorder) Submit(submission promptRemoteSubmission) (string, error) {
	r.submitCalls++
	r.submissions = append(r.submissions, submission)
	if r.submitErr != nil {
		return "", r.submitErr
	}
	return "ralph-prompt-abcde", nil
}

func (r *promptRemoteRecorder) PrintLogHint(workflowName string) {
	r.hintCalls++
	r.hints = append(r.hints, workflowName)
}

func (r *promptRemoteRecorder) FollowLogs(workflowName string) error {
	r.followLogsCnt++
	r.followed = append(r.followed, workflowName)
	return r.followErr
}

// promptNotifyRecorder records completion notifications.
type promptNotifyRecorder struct {
	successes []string
	errors    []string
}

func (n *promptNotifyRecorder) Success(slug string) {
	n.successes = append(n.successes, slug)
}

func (n *promptNotifyRecorder) Error(slug string) {
	n.errors = append(n.errors, slug)
}

// TestPromptCmdRemoteSubmission covers remote mode resolving the slug, prompt
// branch, base branch, and prompt locally, checking the starting branch
// against the remote, and submitting those resolved values.
func TestPromptCmdRemoteSubmission(t *testing.T) {
	var synced []string
	remote := &promptRemoteRecorder{}
	cmd := &PromptCmd{
		Prompt:       []string{"do", "work"},
		Mode:         config.ModeRemote,
		NoFollow:     true,
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		remote:       remote,
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) { return "main", nil },
			IsBranchSyncedWithRemoteFunc: func(branch string) error {
				synced = append(synced, branch)
				return nil
			},
			SwitchToBranchFunc: func(string) error {
				t.Fatal("remote mode must not switch the local checkout")
				return nil
			},
		},
		worktree: &promptWorktreeRecorder{},
	}
	preparePromptRun(cmd)

	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{"main"}, synced, "the starting branch is checked against the remote")
	require.Equal(t, 1, remote.submitCalls)
	assert.Equal(t, promptRemoteSubmission{
		CloneBranch:  "main",
		PromptBranch: "my-feature",
		BaseBranch:   "main",
		Prompt:       "do work",
	}, remote.submissions[0])
	assert.Empty(t, cmd.worktree.(*promptWorktreeRecorder).created, "remote mode creates no local worktree")
}

// TestPromptCmdRemoteBranchSyncRejection covers a starting branch missing from
// the remote or out of sync: submission fails before a workflow is created.
func TestPromptCmdRemoteBranchSyncRejection(t *testing.T) {
	syncErr := errors.New("branch 'main' has not been pushed to remote")
	remote := &promptRemoteRecorder{}
	cmd := &PromptCmd{
		Prompt:       []string{"do work"},
		Mode:         config.ModeRemote,
		NoFollow:     true,
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
		remote:       remote,
		git: &git.MockClient{
			CurrentBranchFunc:            func() (string, error) { return "main", nil },
			IsBranchSyncedWithRemoteFunc: func(string) error { return syncErr },
		},
	}

	err := cmd.Run()
	require.ErrorIs(t, err, syncErr)
	assert.Zero(t, remote.submitCalls, "a rejected branch submits no workflow")
}

// TestPromptCmdRemoteReadsPromptLocally covers a prompt read from a file and
// from standard input before submission, sent as text to the remote workflow.
func TestPromptCmdRemoteReadsPromptLocally(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(path, []byte("from the file\n"), 0644))

	tests := []struct {
		name string
		cmd  *PromptCmd
		want string
	}{
		{
			name: "file prompt",
			cmd:  &PromptCmd{File: path},
			want: "from the file\n",
		},
		{
			name: "standard input prompt",
			cmd: &PromptCmd{
				stdin:           strings.NewReader("from standard input"),
				stdinIsTerminal: func() bool { return false },
			},
			want: "from standard input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := &promptRemoteRecorder{}
			cmd := tt.cmd
			cmd.Mode = config.ModeRemote
			cmd.NoFollow = true
			cmd.config = &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }}
			cmd.slugProposer = promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil })
			cmd.remote = remote
			cmd.git = &git.MockClient{CurrentBranchFunc: func() (string, error) { return "main", nil }}
			preparePromptRun(cmd)

			require.NoError(t, cmd.Run())
			require.Equal(t, 1, remote.submitCalls)
			assert.Equal(t, tt.want, remote.submissions[0].Prompt)
		})
	}
}

// TestPromptCmdRemoteFollow covers the default follow and the no-follow hint,
// including the completion notification.
func TestPromptCmdRemoteFollow(t *testing.T) {
	tests := []struct {
		name         string
		noFollow     bool
		followErr    error
		wantHint     bool
		wantFollow   bool
		wantSuccess  []string
		wantErrors   []string
		wantRunError bool
	}{
		{
			name:        "remote follows by default and notifies success",
			wantFollow:  true,
			wantSuccess: []string{"my-feature"},
		},
		{
			name:       "no-follow prints the log hint",
			noFollow:   true,
			wantHint:   true,
			wantFollow: false,
		},
		{
			name:         "a followed failure notifies the error",
			followErr:    errors.New("workflow failed"),
			wantFollow:   true,
			wantErrors:   []string{"my-feature"},
			wantRunError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := &promptRemoteRecorder{followErr: tt.followErr}
			notifier := &promptNotifyRecorder{}
			cmd := &PromptCmd{
				Prompt:       []string{"do work"},
				Mode:         config.ModeRemote,
				NoFollow:     tt.noFollow,
				config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "my-feature", nil }),
				remote:       remote,
				notify:       notifier,
				git:          &git.MockClient{CurrentBranchFunc: func() (string, error) { return "main", nil }},
			}
			preparePromptRun(cmd)

			err := cmd.Run()
			if tt.wantRunError {
				require.ErrorIs(t, err, tt.followErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantHint, remote.hintCalls > 0)
			assert.Equal(t, tt.wantFollow, remote.followLogsCnt > 0)
			assert.Equal(t, tt.wantSuccess, notifier.successes)
			assert.Equal(t, tt.wantErrors, notifier.errors)
		})
	}
}

// promptIndex returns the position of event in the recorded order, or -1.
func promptIndex(order []string, event string) int {
	for i, got := range order {
		if got == event {
			return i
		}
	}
	return -1
}

// promptLastIndex returns the position of the last occurrence of event, or -1.
func promptLastIndex(order []string, event string) int {
	for i := len(order) - 1; i >= 0; i-- {
		if order[i] == event {
			return i
		}
	}
	return -1
}
