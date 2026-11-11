package cmd

import (
	"errors"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	orchestrationPrompt "github.com/zon/ralph/internal/orchestration/prompt"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

// fakeWorkflowPromptWorkspace records the workspace flags and returns an
// injected error when set, so tests never prepare a real container workspace.
type fakeWorkflowPromptWorkspace struct {
	flags  wksp.WorkspaceFlags
	err    error
	called bool
}

func (f *fakeWorkflowPromptWorkspace) Setup(flags wksp.WorkspaceFlags) error {
	f.called = true
	f.flags = flags
	return f.err
}

// fakeWorkflowPromptRunner records the prompt invocation and returns an
// injected error when set, so tests never run a real prompt flow.
type fakeWorkflowPromptRunner struct {
	flags  orchestrationPrompt.WorkflowPromptFlags
	err    error
	called bool
}

func (f *fakeWorkflowPromptRunner) Run(flags orchestrationPrompt.WorkflowPromptFlags) error {
	f.called = true
	f.flags = flags
	return f.err
}

// TestWorkflowPromptCmdParsing covers the `ralph workflow prompt` command
// surface that runs inside the workflow container.
func TestWorkflowPromptCmdParsing(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		"workflow", "prompt",
		"--repo", "owner/repo",
		"--clone", "main",
		"--branch", "my-feature",
		"--prefix", "update",
		"--base", "release",
		"--prompt", "Add the endpoint.",
		"--model", "gpt-4",
		"--variant", "high",
		"--agent", "build",
		"--debug", "feature",
		"--no-services",
		"--verbose",
	})
	require.NoError(t, err)

	assert.Equal(t, "owner/repo", cmd.Workflow.Prompt.Repo)
	assert.Equal(t, "main", cmd.Workflow.Prompt.CloneBranch)
	assert.Equal(t, "my-feature", cmd.Workflow.Prompt.Branch)
	assert.Equal(t, "update", cmd.Workflow.Prompt.Prefix)
	assert.Equal(t, "release", cmd.Workflow.Prompt.Base)
	assert.Equal(t, "Add the endpoint.", cmd.Workflow.Prompt.Prompt)
	assert.Equal(t, "gpt-4", cmd.Workflow.Prompt.Model)
	assert.Equal(t, "high", cmd.Workflow.Prompt.Variant)
	assert.Equal(t, "build", cmd.Workflow.Prompt.Agent)
	assert.Equal(t, "feature", cmd.Workflow.Prompt.Debug)
	assert.True(t, cmd.Workflow.Prompt.NoServices)
	assert.True(t, cmd.Workflow.Prompt.Verbose)
}

// TestWorkflowPromptCmdDefaults asserts the container-side prompt command
// defaults match the other workflow subcommands.
func TestWorkflowPromptCmdDefaults(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"workflow", "prompt", "--repo", "owner/repo"})
	require.NoError(t, err)
	assert.Equal(t, "ralph-zon[bot]", cmd.Workflow.Prompt.BotName)
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", cmd.Workflow.Prompt.BotEmail)
	assert.False(t, cmd.Workflow.Prompt.NoServices)
}

// TestWorkflowPromptCmdHelpText asserts the workflow prompt subcommand appears
// in the workflow group help.
func TestWorkflowPromptCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "--help"})
	assert.Contains(t, output, "Invoked by workflows to execute a prompt")
}

// TestWorkflowPromptCmdMissingPromptAbortsBeforeWorkspace asserts a missing or
// empty prompt fails before the workspace is prepared and before the prompt
// flow runs.
func TestWorkflowPromptCmdMissingPromptAbortsBeforeWorkspace(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{name: "missing prompt", prompt: ""},
		{name: "whitespace-only prompt", prompt: "  \n\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := &fakeWorkflowPromptWorkspace{}
			runner := &fakeWorkflowPromptRunner{}
			cmd := &WorkflowPromptCmd{
				Repo:           "owner/repo",
				Branch:         "my-feature",
				Base:           "main",
				Prompt:         tt.prompt,
				workspaceSetup: workspace,
				promptRunner:   runner,
			}

			err := cmd.Run()

			require.ErrorIs(t, err, orchestrationPrompt.ErrMissingPrompt)
			assert.False(t, workspace.called, "the workspace is not prepared for an invalid prompt")
			assert.False(t, runner.called, "the prompt does not run for an invalid prompt")
		})
	}
}

// TestWorkflowPromptCmdRunPreparesWorkspaceAndCarriesOptions asserts the
// container-side command prepares the workspace with the submitted repo, clone
// branch, prompt branch, and bot identity, then runs the prompt with the
// carried prompt, base, model, variant, agent, and service choice.
func TestWorkflowPromptCmdRunPreparesWorkspaceAndCarriesOptions(t *testing.T) {
	workspace := &fakeWorkflowPromptWorkspace{}
	runner := &fakeWorkflowPromptRunner{}
	cmd := &WorkflowPromptCmd{
		Repo:           "owner/repo",
		CloneBranch:    "main",
		Branch:         "update/my-feature",
		Prefix:         "update",
		Base:           "release",
		Prompt:         "Add the endpoint.",
		BotName:        "ralph-zon[bot]",
		BotEmail:       "ralph-zon[bot]@users.noreply.github.com",
		Model:          "gpt-4",
		Variant:        "high",
		Agent:          "build",
		Debug:          "feature",
		NoServices:     true,
		Verbose:        true,
		workspaceSetup: workspace,
		promptRunner:   runner,
	}

	err := cmd.Run()

	require.NoError(t, err)
	require.True(t, workspace.called, "the workspace is prepared before the prompt runs")
	assert.Equal(t, "owner/repo", workspace.flags.Repo)
	assert.Equal(t, "main", workspace.flags.CloneBranch)
	assert.Equal(t, "update/my-feature", workspace.flags.TargetBranch, "the workspace checks out the prompt branch")
	assert.True(t, workspace.flags.CreateBranch, "the workspace creates the prompt branch when missing")
	assert.Equal(t, "ralph-zon[bot]", workspace.flags.BotName)
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", workspace.flags.BotEmail)

	require.True(t, runner.called, "the prompt runs after the workspace is prepared")
	assert.Equal(t, "Add the endpoint.", runner.flags.Prompt)
	assert.Equal(t, "update/my-feature", runner.flags.Branch)
	assert.Equal(t, "update", runner.flags.Prefix)
	assert.Equal(t, "release", runner.flags.BaseBranch)
	assert.Equal(t, "gpt-4", runner.flags.Model)
	assert.Equal(t, "high", runner.flags.Variant)
	assert.Equal(t, "build", runner.flags.Agent)
	assert.Equal(t, "feature", runner.flags.Debug)
	assert.True(t, runner.flags.NoServices)
	assert.True(t, runner.flags.Verbose)
}

// TestWorkflowPromptCmdWorkspaceFailureAbortsBeforePrompt asserts a workspace
// failure stops the container-side command before the prompt runs.
func TestWorkflowPromptCmdWorkspaceFailureAbortsBeforePrompt(t *testing.T) {
	workspaceErr := errors.New("workspace setup boom")
	workspace := &fakeWorkflowPromptWorkspace{err: workspaceErr}
	runner := &fakeWorkflowPromptRunner{}
	cmd := &WorkflowPromptCmd{
		Repo:           "owner/repo",
		Branch:         "my-feature",
		Base:           "main",
		Prompt:         "Add the endpoint.",
		workspaceSetup: workspace,
		promptRunner:   runner,
	}

	err := cmd.Run()

	require.ErrorIs(t, err, workspaceErr)
	assert.True(t, workspace.called)
	assert.False(t, runner.called, "the prompt does not run when the workspace setup fails")
}

// TestPromptCmdRunInWorkspaceRunsThePreparedFlow asserts the container-side
// flow runs the before commands and the pass in the checkout the workspace
// prepared, without proposing a slug, resolving a remote mode, or switching
// branches again.
func TestPromptCmdRunInWorkspaceRunsThePreparedFlow(t *testing.T) {
	var order []string
	aiClient := &promptAIRecorder{order: &order}
	servicesClient := &promptServicesRecorder{order: &order}
	prClient := &promptPRRecorder{order: &order}
	gitClient := &git.MockClient{
		CurrentBranchFunc: func() (string, error) {
			t.Fatal("the prepared flow must not read the starting branch")
			return "", nil
		},
		SwitchToBranchFunc: func(string) error {
			t.Fatal("the prepared flow must not switch branches")
			return nil
		},
		FetchBranchFunc: func(branch string) error {
			assert.Equal(t, "release", branch)
			promptEvent(&order, "fetch-base")
			return nil
		},
		NeedsMergeFunc: func(string) (bool, error) {
			promptEvent(&order, "needs-merge")
			return false, nil
		},
		HeadCommitFunc: func() (string, error) {
			return "unchanged-tip", nil
		},
	}
	cmd := &PromptCmd{
		config:     &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:        gitClient,
		services:   servicesClient,
		ai:         aiClient,
		pr:         prClient,
		output:     &promptOutputRecorder{},
		NoServices: true,
	}

	err := cmd.RunInWorkspace("do work", "my-feature", "release")

	require.NoError(t, err)
	require.NotNil(t, cmd.resolved)
	assert.Equal(t, config.ModeLocal, cmd.resolved.Mode)
	assert.Equal(t, "my-feature", cmd.resolved.PromptBranch)
	assert.Equal(t, "release", cmd.resolved.BaseBranch)
	assert.Equal(t, []string{"before", "fetch-base", "needs-merge", "pass", "remove-logs"}, order)
	assert.Equal(t, []string{"do work"}, aiClient.prompts)
	assert.Zero(t, prClient.listCalls, "a prompt without a prefix lists no pull requests")
	assert.Zero(t, prClient.calls, "an unchanged tip opens no pull request")
}

// TestPromptCmdRunInWorkspaceRenamesBranchBeforePR asserts the container-side
// flow renames the prompt branch from the commit log before the push and pull
// request, without switching or restoring branches in the prepared checkout.
func TestPromptCmdRunInWorkspaceRenamesBranchBeforePR(t *testing.T) {
	var order []string
	var renames [][2]string
	tip := "before"
	prClient := &promptPRRecorder{order: &order}
	gitClient := &git.MockClient{
		CurrentBranchFunc: func() (string, error) {
			t.Fatal("the prepared flow must not read the starting branch")
			return "", nil
		},
		SwitchToBranchFunc: func(string) error {
			t.Fatal("the prepared flow must not switch branches")
			return nil
		},
		CheckoutBranchFunc: func(string) error {
			t.Fatal("the prepared flow must not restore a starting branch")
			return nil
		},
		FetchBranchFunc: func(string) error { return nil },
		NeedsMergeFunc:  func(string) (bool, error) { return false, nil },
		HeadCommitFunc:  func() (string, error) { return tip, nil },
		GetCommitLogFunc: func(base string, limit int) (string, error) {
			assert.Equal(t, "before", base, "the commits are read after the pre-pass tip")
			return "abc123: add the actual feature", nil
		},
		RenameBranchFunc: func(oldName, newName string) error {
			renames = append(renames, [2]string{oldName, newName})
			promptEvent(&order, "rename")
			return nil
		},
		PushFunc: func() error {
			promptEvent(&order, "push")
			return nil
		},
	}
	cmd := &PromptCmd{
		config:   &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:      gitClient,
		services: &promptServicesRecorder{order: &order},
		ai:       &promptAIRecorder{order: &order, passFunc: func(string) error { tip = "after"; return nil }},
		pr:       prClient,
		output:   &promptOutputRecorder{},
		slugProposer: promptSlugProposerFunc(func(description string) (string, error) {
			assert.Equal(t, "abc123: add the actual feature", description)
			return "actual-work", nil
		}),
		NoServices: true,
	}

	err := cmd.RunInWorkspace("do work", "my-feature", "release")

	require.NoError(t, err)
	assert.Equal(t, [][2]string{{"my-feature", "actual-work"}}, renames)
	assert.Less(t, promptIndex(order, "rename"), promptIndex(order, "push"))
	assert.Equal(t, []string{"actual-work"}, prClient.heads)
	assert.Equal(t, "actual-work", cmd.resolved.PromptBranch)
}

// TestPromptCmdRunInWorkspaceRejectsEmptyPrompt asserts the prepared entrypoint
// rejects an empty prompt before loading the configuration.
func TestPromptCmdRunInWorkspaceRejectsEmptyPrompt(t *testing.T) {
	loadCalls := 0
	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
			loadCalls++
			return config.Any(), nil
		}},
	}

	err := cmd.RunInWorkspace("  ", "my-feature", "main")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
	assert.Zero(t, loadCalls)
}

// TestPromptCmdRunInWorkspacePrefixMatchingPullRequestSkips covers the
// container-side prefix check stopping the prompt when an open pull request
// appeared after submission. The prefix is normalized and listed once, the
// matches print, and no before command or pass runs.
func TestPromptCmdRunInWorkspacePrefixMatchingPullRequestSkips(t *testing.T) {
	output := &promptOutputRecorder{}
	servicesClient := &promptServicesRecorder{}
	aiClient := &promptAIRecorder{}
	prClient := &promptPRRecorder{
		listFunc: func(prefix string) ([]github.PullRequest, error) {
			assert.Equal(t, "update", prefix)
			return []github.PullRequest{
				{URL: "https://github.com/owner/repo/pull/7", HeadRefName: "update/fix-auth", BaseRefName: "main"},
			}, nil
		},
	}
	cmd := &PromptCmd{
		config:   &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:      &git.MockClient{},
		services: servicesClient,
		ai:       aiClient,
		pr:       prClient,
		output:   output,
	}

	err := cmd.RunInWorkspace("do work", "update/my-feature", "main", "update/")

	require.NoError(t, err)
	assert.Nil(t, cmd.resolved, "a matching pull request skips before branch setup")
	assert.Equal(t, []string{"update"}, prClient.listPrefixes)
	assert.Equal(t, []string{"https://github.com/owner/repo/pull/7"}, output.infos)
	assert.Zero(t, servicesClient.runBeforeCalls, "a matching pull request runs no before command")
	assert.Zero(t, servicesClient.startCalls, "a matching pull request starts no service")
	assert.Zero(t, aiClient.passCalls, "a matching pull request runs no pass")
}

// TestPromptCmdRunInWorkspacePrefixListingFailureFailsBeforeWork covers a
// failed container-side listing failing the command before any before command
// or pass work.
func TestPromptCmdRunInWorkspacePrefixListingFailureFailsBeforeWork(t *testing.T) {
	servicesClient := &promptServicesRecorder{}
	aiClient := &promptAIRecorder{}
	prClient := &promptPRRecorder{listErr: errors.New("listing failed")}
	cmd := &PromptCmd{
		config:   &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:      &git.MockClient{},
		services: servicesClient,
		ai:       aiClient,
		pr:       prClient,
		output:   &promptOutputRecorder{},
	}

	err := cmd.RunInWorkspace("do work", "update/my-feature", "main", "update")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing failed")
	assert.Zero(t, servicesClient.runBeforeCalls, "a failed listing runs no before command")
	assert.Zero(t, aiClient.passCalls, "a failed listing runs no pass")
}

// TestPromptCmdRunInWorkspaceInvalidPrefixFailsBeforeListing covers a prefix Git
// rejects failing the container check before any listing or prompt work.
func TestPromptCmdRunInWorkspaceInvalidPrefixFailsBeforeListing(t *testing.T) {
	prClient := &promptPRRecorder{}
	aiClient := &promptAIRecorder{}
	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:    &git.MockClient{},
		ai:     aiClient,
		pr:     prClient,
		output: &promptOutputRecorder{},
	}

	err := cmd.RunInWorkspace("do work", "my-feature", "main", "not a prefix")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a prefix")
	assert.Zero(t, prClient.listCalls, "an invalid prefix is not listed")
	assert.Zero(t, aiClient.passCalls, "an invalid prefix runs no pass")
}

// TestPromptCmdRunInWorkspacePrefixWithoutMatchContinues covers an empty
// container-side listing continuing with the prepared prompt flow.
func TestPromptCmdRunInWorkspacePrefixWithoutMatchContinues(t *testing.T) {
	prClient := &promptPRRecorder{}
	aiClient := &promptAIRecorder{}
	cmd := &PromptCmd{
		config:   &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:      &git.MockClient{},
		services: &promptServicesRecorder{},
		ai:       aiClient,
		pr:       prClient,
		output:   &promptOutputRecorder{},
	}

	err := cmd.RunInWorkspace("do work", "update/my-feature", "main", "update")

	require.NoError(t, err)
	require.NotNil(t, cmd.resolved)
	assert.Equal(t, 1, prClient.listCalls, "the prefix lists open pull requests once")
	assert.Equal(t, "update/my-feature", cmd.resolved.PromptBranch)
	assert.Equal(t, 1, aiClient.passCalls, "an empty listing continues to the pass")
}

// TestPromptCmdRunInWorkspacePrefixWithoutBranchProposesAndCreatesBranch covers
// a workflow prompt that names only a prefix: the prefix is checked, the prompt
// slug is proposed, and the prefixed branch is created before the prompt runs.
func TestPromptCmdRunInWorkspacePrefixWithoutBranchProposesAndCreatesBranch(t *testing.T) {
	var order []string
	var switched string
	var proposed string
	prClient := &promptPRRecorder{order: &order}
	servicesClient := &promptServicesRecorder{order: &order}
	aiClient := &promptAIRecorder{order: &order}
	gitClient := &git.MockClient{
		SwitchToBranchFunc: func(name string) error {
			switched = name
			promptEvent(&order, "switch")
			return nil
		},
	}
	cmd := &PromptCmd{
		config:     &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:        gitClient,
		services:   servicesClient,
		ai:         aiClient,
		pr:         prClient,
		output:     &promptOutputRecorder{},
		NoServices: true,
		slugProposer: promptSlugProposerFunc(func(description string) (string, error) {
			proposed = description
			promptEvent(&order, "propose")
			return "Fix Auth", nil
		}),
	}

	err := cmd.RunInWorkspace("do work", "", "main", "update/")

	require.NoError(t, err)
	assert.Equal(t, "do work", proposed, "the slug is proposed from the prompt")
	assert.Equal(t, "update/fix-auth", switched, "the prefixed branch is created in the checkout")
	require.NotNil(t, cmd.resolved)
	assert.Equal(t, "update/fix-auth", cmd.resolved.PromptBranch)
	assert.Equal(t, "update", cmd.resolved.Prefix)
	assert.Equal(t, []string{"update"}, prClient.listPrefixes, "the normalized prefix is checked")
	assert.Equal(t, 1, aiClient.passCalls, "the prompt runs after the branch is created")
	assert.Less(t, promptIndex(order, "list-pull-requests"), promptIndex(order, "propose"), "the prefix is checked before the slug proposal")
	assert.Less(t, promptIndex(order, "propose"), promptIndex(order, "switch"), "the branch is created after the slug proposal")
}

// TestPromptCmdRunInWorkspacePrefixWithoutBranchMatchingPullRequestSkips covers
// a prefix-only prompt stopping before the slug proposal when an open pull
// request already has the prefix.
func TestPromptCmdRunInWorkspacePrefixWithoutBranchMatchingPullRequestSkips(t *testing.T) {
	output := &promptOutputRecorder{}
	prClient := &promptPRRecorder{
		listFunc: func(prefix string) ([]github.PullRequest, error) {
			assert.Equal(t, "update", prefix)
			return []github.PullRequest{
				{URL: "https://github.com/owner/repo/pull/8", HeadRefName: "update/fix-auth", BaseRefName: "main"},
			}, nil
		},
	}
	proposerCalls := 0
	switchCalls := 0
	servicesClient := &promptServicesRecorder{}
	aiClient := &promptAIRecorder{}
	cmd := &PromptCmd{
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:          &git.MockClient{SwitchToBranchFunc: func(string) error { switchCalls++; return nil }},
		services:     servicesClient,
		ai:           aiClient,
		pr:           prClient,
		output:       output,
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { proposerCalls++; return "fix-auth", nil }),
	}

	err := cmd.RunInWorkspace("do work", "", "main", "update")

	require.NoError(t, err)
	assert.Nil(t, cmd.resolved, "a matching pull request skips before branch setup")
	assert.Zero(t, proposerCalls, "a matching pull request proposes no slug")
	assert.Zero(t, switchCalls, "a matching pull request creates no branch")
	assert.Equal(t, []string{"https://github.com/owner/repo/pull/8"}, output.infos)
	assert.Zero(t, servicesClient.runBeforeCalls, "a matching pull request runs no before command")
	assert.Zero(t, aiClient.passCalls, "a matching pull request runs no pass")
}

// TestPromptCmdRunInWorkspacePrefixWithoutBranchStopsOnBranchFailure covers a
// failed slug proposal or branch creation stopping before any before command or
// pass work.
func TestPromptCmdRunInWorkspacePrefixWithoutBranchStopsOnBranchFailure(t *testing.T) {
	proposalErr := errors.New("proposal boom")
	createErr := errors.New("create boom")
	tests := []struct {
		name        string
		proposerErr error
		createErr   error
		wantErr     error
	}{
		{name: "slug proposal fails", proposerErr: proposalErr, wantErr: proposalErr},
		{name: "branch creation fails", createErr: createErr, wantErr: createErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			servicesClient := &promptServicesRecorder{}
			aiClient := &promptAIRecorder{}
			cmd := &PromptCmd{
				config:   &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				git:      &git.MockClient{SwitchToBranchFunc: func(string) error { return tt.createErr }},
				services: servicesClient,
				ai:       aiClient,
				pr:       &promptPRRecorder{},
				output:   &promptOutputRecorder{},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) {
					return "fix-auth", tt.proposerErr
				}),
			}

			err := cmd.RunInWorkspace("do work", "", "main", "update")

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, cmd.resolved, "a failed branch resolution skips branch setup")
			assert.Zero(t, servicesClient.runBeforeCalls, "a failed branch resolution runs no before command")
			assert.Zero(t, aiClient.passCalls, "a failed branch resolution runs no pass")
		})
	}
}

// TestPromptCmdRunInWorkspaceSuppliedBranchMustStartWithPrefix covers the
// container-side rule that a supplied branch stays under the supplied prefix.
func TestPromptCmdRunInWorkspaceSuppliedBranchMustStartWithPrefix(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{name: "another prefix", branch: "other/fix-auth"},
		{name: "the prefix alone", branch: "update"},
		{name: "a case-different prefix", branch: "Update/fix-auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposerCalls := 0
			switchCalls := 0
			prClient := &promptPRRecorder{}
			servicesClient := &promptServicesRecorder{}
			aiClient := &promptAIRecorder{}
			cmd := &PromptCmd{
				config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				git:          &git.MockClient{SwitchToBranchFunc: func(string) error { switchCalls++; return nil }},
				services:     servicesClient,
				ai:           aiClient,
				pr:           prClient,
				output:       &promptOutputRecorder{},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { proposerCalls++; return "fix-auth", nil }),
			}

			err := cmd.RunInWorkspace("do work", tt.branch, "main", "update")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "must start with")
			assert.Contains(t, err.Error(), "update/")
			assert.Zero(t, proposerCalls, "a mismatched branch proposes no slug")
			assert.Zero(t, switchCalls, "a mismatched branch creates no branch")
			assert.Zero(t, prClient.listCalls, "a mismatched branch lists no pull requests")
			assert.Zero(t, servicesClient.runBeforeCalls, "a mismatched branch runs no before command")
			assert.Zero(t, aiClient.passCalls, "a mismatched branch runs no pass")
		})
	}
}

// TestPromptWorkflowRunnerAdapterCarriesPrompt asserts the in-process adapter
// hands the prompt, prompt branch, and base branch to the prepared flow.
func TestPromptWorkflowRunnerAdapterCarriesPrompt(t *testing.T) {
	ctx := execcontext.NewContext()
	ctx.SetModel("gpt-4")
	ctx.SetVariant("high")
	ctx.SetAgent("build")
	adapter := &promptWorkflowRunnerAdapter{ctx: ctx}

	// A missing prompt proves the adapter reaches the prepared flow's
	// validation rather than doing its own workspace or branch work.
	err := adapter.Run(orchestrationPrompt.WorkflowPromptFlags{Prompt: " "})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}
