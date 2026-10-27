package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
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
	checked     []string
	created     []string
	removed     []string
	createCalls int
}

func (w *promptWorktreeRecorder) CreateWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error) {
	w.created = append(w.created, branch)
	w.createCalls++
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
	if w.checkErr != nil {
		return nil, false, w.checkErr
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "list", "--porcelain"}}, w.checkedOut, nil
}

func (w *promptWorktreeRecorder) RemoveWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error) {
	w.removed = append(w.removed, branch)
	if w.removeErr != nil {
		return nil, w.removeErr
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "remove", branch}}, nil
}

// promptWorkspaceRecorder records directory changes so worktree cleanup can be
// asserted without moving the test process.
type promptWorkspaceRecorder struct {
	dirs []string
	err  error
}

func (w *promptWorkspaceRecorder) ChangeDirectory(path string) error {
	w.dirs = append(w.dirs, path)
	return w.err
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

	require.NoError(t, cmd.Run())
	assert.Equal(t, []string{"my-feature"}, worktree.checked)
	assert.Equal(t, []string{"my-feature"}, worktree.created)
	assert.Equal(t, []string{"my-feature"}, worktree.removed)
	assert.Equal(t, []string{"/worktree/my-feature", originalDir}, workspace.dirs)
	assert.Zero(t, switchCalls, "the starting checkout stays on its original branch")
	assert.Equal(t, "main", cmd.resolved.CurrentBranch)
}
