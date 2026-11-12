package cmd

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	orchestrationPrompt "github.com/zon/ralph/internal/orchestration/prompt"
	"github.com/zon/ralph/internal/output"
)

// TestPromptCmdRunRemoteFromCloneSubmits covers a target prompt run from the
// prepared clone: the prefix is checked, the base branch is resolved, the clone
// branch is checked against the remote, and one workflow is submitted. The
// prompt slug is left to the target workflow, so the launcher pod asks the AI
// for nothing. No local pass or service work runs.
func TestPromptCmdRunRemoteFromCloneSubmits(t *testing.T) {
	var order []string
	var synced []string
	var proposals int
	remote := &promptRemoteRecorder{}
	aiClient := &promptAIRecorder{order: &order}
	servicesClient := &promptServicesRecorder{order: &order}
	prClient := &promptPRRecorder{order: &order}
	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			CurrentBranchFunc: func() (string, error) {
				promptEvent(&order, "current")
				return "main", nil
			},
			IsBranchSyncedWithRemoteFunc: func(branch string) error {
				synced = append(synced, branch)
				promptEvent(&order, "sync")
				return nil
			},
		},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) {
			proposals++
			return "Fix Auth", nil
		}),
		services: servicesClient,
		ai:       aiClient,
		pr:       prClient,
		output:   &promptOutputRecorder{},
		remote:   remote,
		notify:   &promptNotifyRecorder{},
		NoNotify: true,
	}

	err := cmd.RunRemoteFromClone("do work", "", "release", "update/")

	require.NoError(t, err)
	assert.Zero(t, proposals, "the launcher pod proposes no slug")
	assert.Equal(t, []string{"main"}, synced, "the clone branch is checked against the remote")
	require.Equal(t, 1, remote.submitCalls, "exactly one target workflow is submitted")
	assert.Equal(t, promptRemoteSubmission{
		CloneBranch: "main",
		Prefix:      "update",
		BaseBranch:  "release",
		Prompt:      "do work",
	}, remote.submissions[0], "a prefix alone is sent with no prompt branch")
	require.NotNil(t, cmd.resolved)
	assert.Equal(t, config.ModeRemote, cmd.resolved.Mode)
	assert.Empty(t, cmd.resolved.PromptBranch)
	assert.Equal(t, "update", cmd.resolved.Prefix)
	assert.Equal(t, "release", cmd.resolved.BaseBranch)
	assert.Less(t, promptIndex(order, "list-pull-requests"), promptIndex(order, "current"), "the prefix is checked before the branches resolve")
	assert.Less(t, promptIndex(order, "current"), promptIndex(order, "sync"), "the base is resolved before the sync check")
	assert.Zero(t, servicesClient.runBeforeCalls, "a target run runs no local before command")
	assert.Zero(t, servicesClient.startCalls, "a target run starts no local service")
	assert.Zero(t, aiClient.passCalls, "a target run performs no local pass")
}

// TestPromptCmdRunRemoteFromCloneMatchingPullRequestSkips covers a prefix that
// already has an open pull request: the matches print, no workflow is
// submitted, and no branch or prompt work starts.
func TestPromptCmdRunRemoteFromCloneMatchingPullRequestSkips(t *testing.T) {
	var proposed, synced int
	output := &promptOutputRecorder{}
	remote := &promptRemoteRecorder{}
	prClient := &promptPRRecorder{
		listFunc: func(prefix string) ([]github.PullRequest, error) {
			assert.Equal(t, "update", prefix)
			return []github.PullRequest{
				{URL: "https://github.com/owner/repo/pull/7", HeadRefName: "update/fix-auth", BaseRefName: "main"},
			}, nil
		},
	}
	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			IsBranchSyncedWithRemoteFunc: func(string) error { synced++; return nil },
		},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { proposed++; return "fix-auth", nil }),
		pr:           prClient,
		output:       output,
		remote:       remote,
	}

	err := cmd.RunRemoteFromClone("do work", "", "main", "update")

	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, prClient.listPrefixes)
	assert.Equal(t, []string{"https://github.com/owner/repo/pull/7"}, output.infos)
	assert.Zero(t, proposed, "a matching pull request proposes no slug")
	assert.Zero(t, synced, "a matching pull request checks no branch sync")
	assert.Zero(t, remote.submitCalls, "a matching pull request submits no workflow")
	assert.Nil(t, cmd.resolved)
}

// TestPromptCmdRunRemoteFromClonePrefixListingFailure covers a failed prefix
// listing failing before any branch, sync, or submission work.
func TestPromptCmdRunRemoteFromClonePrefixListingFailure(t *testing.T) {
	listErr := errors.New("listing failed")
	remote := &promptRemoteRecorder{}
	cmd := &PromptCmd{
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:          &git.MockClient{},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "fix-auth", nil }),
		pr:           &promptPRRecorder{listErr: listErr},
		output:       &promptOutputRecorder{},
		remote:       remote,
	}

	err := cmd.RunRemoteFromClone("do work", "", "main", "update")

	require.ErrorIs(t, err, listErr)
	assert.Zero(t, remote.submitCalls)
}

// TestPromptCmdRunRemoteFromCloneRejectsBranchOutsidePrefix covers a supplied
// branch that is not under the prefix failing before any listing or submission.
func TestPromptCmdRunRemoteFromCloneRejectsBranchOutsidePrefix(t *testing.T) {
	prClient := &promptPRRecorder{}
	remote := &promptRemoteRecorder{}
	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:    &git.MockClient{},
		pr:     prClient,
		output: &promptOutputRecorder{},
		remote: remote,
	}

	err := cmd.RunRemoteFromClone("do work", "other/fix-auth", "main", "update")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with")
	assert.Zero(t, prClient.listCalls, "a mismatched branch lists no pull requests")
	assert.Zero(t, remote.submitCalls, "a mismatched branch submits no workflow")
}

// TestPromptCmdRunRemoteFromCloneSyncRejection covers a clone branch missing
// from the remote: the command fails before a workflow is submitted.
func TestPromptCmdRunRemoteFromCloneSyncRejection(t *testing.T) {
	syncErr := errors.New("branch 'main' has not been pushed to remote")
	remote := &promptRemoteRecorder{}
	cmd := &PromptCmd{
		config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:          &git.MockClient{IsBranchSyncedWithRemoteFunc: func(string) error { return syncErr }},
		slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "fix-auth", nil }),
		pr:           &promptPRRecorder{},
		output:       &promptOutputRecorder{},
		remote:       remote,
	}

	err := cmd.RunRemoteFromClone("do work", "update/fix-auth", "main", "update")

	require.ErrorIs(t, err, syncErr)
	assert.Zero(t, remote.submitCalls, "a rejected branch submits no workflow")
}

// TestPromptCmdRunRemoteFromCloneResolvesBase covers the base branch following
// the explicit base, the clone branch when it differs from the prompt branch,
// and the configured default.
func TestPromptCmdRunRemoteFromCloneResolvesBase(t *testing.T) {
	tests := []struct {
		name       string
		base       string
		current    string
		branch     string
		wantBase   string
		wantClone  string
		wantBranch string
	}{
		{
			name:       "explicit base wins",
			base:       "release",
			current:    "main",
			branch:     "update/fix-auth",
			wantBase:   "release",
			wantClone:  "main",
			wantBranch: "update/fix-auth",
		},
		{
			name:       "clone branch supplies the base",
			current:    "develop",
			branch:     "update/fix-auth",
			wantBase:   "develop",
			wantClone:  "develop",
			wantBranch: "update/fix-auth",
		},
		{
			name:       "configured default names the base",
			current:    "update/fix-auth",
			branch:     "update/fix-auth",
			wantBase:   "main",
			wantClone:  "update/fix-auth",
			wantBranch: "update/fix-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := &promptRemoteRecorder{}
			cmd := &PromptCmd{
				config:       &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				git:          &git.MockClient{CurrentBranchFunc: func() (string, error) { return tt.current, nil }},
				slugProposer: promptSlugProposerFunc(func(string) (string, error) { return "fix-auth", nil }),
				pr:           &promptPRRecorder{},
				output:       &promptOutputRecorder{},
				remote:       remote,
			}

			require.NoError(t, cmd.RunRemoteFromClone("do work", tt.branch, tt.base, "update"))

			require.Equal(t, 1, remote.submitCalls)
			assert.Equal(t, tt.wantClone, remote.submissions[0].CloneBranch)
			assert.Equal(t, tt.wantBranch, remote.submissions[0].PromptBranch)
			assert.Equal(t, tt.wantBase, remote.submissions[0].BaseBranch)
		})
	}
}

// TestPromptWorkflowRunnerAdapterTargetNamespaceSubmitsRemote covers the
// workflow runner sending a --namespace prompt through the remote-from-clone
// flow with completion notifications off, instead of running a local pass.
func TestPromptWorkflowRunnerAdapterTargetNamespaceSubmitsRemote(t *testing.T) {
	remote := &promptRemoteRecorder{}
	notifier := &promptNotifyRecorder{}
	aiClient := &promptAIRecorder{}
	servicesClient := &promptServicesRecorder{}
	promptCmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			CurrentBranchFunc:            func() (string, error) { return "main", nil },
			IsBranchSyncedWithRemoteFunc: func(string) error { return nil },
		},
		services: servicesClient,
		ai:       aiClient,
		pr:       &promptPRRecorder{},
		output:   &promptOutputRecorder{},
		remote:   remote,
		notify:   notifier,
	}
	adapter := &promptWorkflowRunnerAdapter{
		ctx:          execcontext.NewContext(),
		newPromptCmd: func(*execcontext.Context) *PromptCmd { return promptCmd },
	}

	err := adapter.Run(orchestrationPrompt.WorkflowPromptFlags{
		CloneBranch: "main",
		Branch:      "update/fix-auth",
		Prefix:      "update",
		BaseBranch:  "release",
		Prompt:      "do work",
		Namespace:   "target-ns",
	})

	require.NoError(t, err)
	require.Equal(t, 1, remote.submitCalls, "the target namespace submits one workflow")
	assert.Equal(t, promptRemoteSubmission{
		CloneBranch:  "main",
		PromptBranch: "update/fix-auth",
		Prefix:       "update",
		BaseBranch:   "release",
		Prompt:       "do work",
	}, remote.submissions[0])
	assert.Zero(t, aiClient.passCalls, "the target namespace runs no pass in this pod")
	assert.Zero(t, servicesClient.runBeforeCalls, "the target namespace runs no local before command")
	assert.Equal(t, []string{"ralph-prompt-abcde"}, remote.followed, "the target workflow logs are followed")
	assert.Zero(t, remote.hintCalls, "a target prompt does not return after submission")
	assert.Empty(t, notifier.successes, "completion notifications are off")
	assert.Empty(t, notifier.errors, "completion notifications are off")
}

// TestPromptWorkflowRunnerAdapterTargetNamespaceFollowsAndPropagates covers the
// target prompt waiting for the workflow to finish: a successful follow exits
// zero, a failed follow exits non-zero, and neither sends a completion
// notification. The target path follows even when the local prompt defaults say
// otherwise.
func TestPromptWorkflowRunnerAdapterTargetNamespaceFollowsAndPropagates(t *testing.T) {
	followErr := errors.New("target workflow failed")
	tests := []struct {
		name      string
		followErr error
	}{
		{name: "target workflow succeeds"},
		{name: "target workflow fails", followErr: followErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := &promptRemoteRecorder{followErr: tt.followErr}
			notifier := &promptNotifyRecorder{}
			promptCmd := &PromptCmd{
				config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
				git: &git.MockClient{
					CurrentBranchFunc:            func() (string, error) { return "main", nil },
					IsBranchSyncedWithRemoteFunc: func(string) error { return nil },
				},
				pr:       &promptPRRecorder{},
				output:   &promptOutputRecorder{},
				remote:   remote,
				notify:   notifier,
				NoFollow: true,
			}
			adapter := &promptWorkflowRunnerAdapter{
				ctx:          execcontext.NewContext(),
				newPromptCmd: func(*execcontext.Context) *PromptCmd { return promptCmd },
			}

			err := adapter.Run(orchestrationPrompt.WorkflowPromptFlags{
				CloneBranch: "main",
				Branch:      "update/fix-auth",
				Prefix:      "update",
				BaseBranch:  "main",
				Prompt:      "do work",
				Namespace:   "target-ns",
			})

			if tt.followErr != nil {
				require.ErrorIs(t, err, tt.followErr, "a failed target workflow exits non-zero")
			} else {
				require.NoError(t, err, "a successful target workflow exits zero")
			}
			assert.Equal(t, []string{"ralph-prompt-abcde"}, remote.followed, "the target workflow logs are followed")
			assert.Zero(t, remote.hintCalls, "the target prompt waits instead of printing a hint")
			assert.Empty(t, notifier.successes, "completion notifications are off")
			assert.Empty(t, notifier.errors, "completion notifications are off")
		})
	}
}

// TestPromptCmdRunRemoteFromCloneFollowsTargetWorkflow covers a target prompt
// following the submitted workflow in the target namespace and leaving it
// running when the follow fails.
func TestPromptCmdRunRemoteFromCloneFollowsTargetWorkflow(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	t.Setenv("KUBECONFIG", t.TempDir()+"/missing-kubeconfig")
	t.Setenv("POD_NAMESPACE", "pod-namespace")

	followErr := errors.New("target workflow failed")
	tests := []struct {
		name      string
		followErr error
	}{
		{name: "target workflow succeeds"},
		{name: "target workflow fails", followErr: followErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logsCtx argo.K8sContext
			var logsWorkflow string
			var logsFollow bool
			stopped := false
			argoClient := &argo.MockClient{
				SubmitYAMLFunc: func(context.Context, string, argo.K8sContext) (string, error) {
					return "ralph-prompt-abcde", nil
				},
				LogsFunc: func(kubeCtx argo.K8sContext, workflowName string, followLogs bool) error {
					logsCtx = kubeCtx
					logsWorkflow = workflowName
					logsFollow = followLogs
					return tt.followErr
				},
				StopWorkflowFunc: func(argo.K8sContext, string) error {
					stopped = true
					return nil
				},
			}

			ctx := execcontext.NewContext()
			ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
			ctx.SetRepoOwner("owner")
			ctx.SetRepoName("repo")
			ctx.SetKubeNamespace("target-ns")

			cfg := config.Any()
			cfg.Workflow.Namespace = "clone-namespace"
			notifier := &promptNotifyRecorder{}
			cmd := &PromptCmd{
				config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return cfg, nil }},
				git: &git.MockClient{
					CurrentBranchFunc:            func() (string, error) { return "main", nil },
					IsBranchSyncedWithRemoteFunc: func(string) error { return nil },
				},
				pr:     &promptPRRecorder{},
				output: &promptOutputRecorder{},
				remote: &promptWorkflowClient{
					ctx:        ctx,
					argoClient: argoClient,
				},
				notify:   notifier,
				NoNotify: true,
			}

			err := cmd.RunRemoteFromClone("do work", "update/fix-auth", "main", "update")

			if tt.followErr != nil {
				require.ErrorIs(t, err, tt.followErr, "a failed target workflow exits non-zero")
			} else {
				require.NoError(t, err, "a successful target workflow exits zero")
			}
			assert.Equal(t, "target-ns", logsCtx.Namespace, "the target workflow logs are followed in the target namespace")
			assert.Empty(t, logsCtx.Name, "the configured context does not apply in a pod")
			assert.Equal(t, "ralph-prompt-abcde", logsWorkflow)
			assert.True(t, logsFollow, "the follow waits for the target workflow")
			assert.False(t, stopped, "following logs leaves the target workflow running")
			assert.Empty(t, notifier.successes, "completion notifications are off")
			assert.Empty(t, notifier.errors, "completion notifications are off")
		})
	}
}

// TestPromptCmdRunRemoteFromCloneUsesCloneConfig covers a target prompt whose
// submitted workflow takes its image, mounts, environment, labels, and
// resources from the config loaded for the clone. The target namespace wins
// over the configured namespace, and the configured context does not apply in a
// pod without a kubeconfig.
func TestPromptCmdRunRemoteFromCloneUsesCloneConfig(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	t.Setenv("KUBECONFIG", t.TempDir()+"/missing-kubeconfig")
	t.Setenv("POD_NAMESPACE", "pod-namespace")

	cfg := config.Any()
	cfg.Workflow = config.WorkflowConfig{
		Image: config.ImageConfig{
			Repository: "registry.example/clone",
			Tag:        "clone-v1",
		},
		ConfigMaps: []config.ConfigMapMount{{Name: "clone-config"}},
		Secrets:    []config.SecretMount{{Name: "clone-secret"}},
		Env:        map[string]config.EnvVar{"CLONE_VAR": {Value: "clone-value"}},
		Context:    "clone-context",
		Namespace:  "clone-namespace",
		Labels:     map[string]string{"team": "clone-team"},
		Resources: config.WorkflowResources{
			Requests: config.ResourceList{CPU: "100m", Memory: "128Mi"},
			Limits:   config.ResourceList{CPU: "500m", Memory: "512Mi"},
		},
	}

	var submittedYAML string
	var submittedCtx argo.K8sContext
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(_ context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submittedYAML = workflowYAML
			submittedCtx = kubeCtx
			return "ralph-prompt-abcde", nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")
	ctx.SetKubeNamespace("target-ns")

	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return cfg, nil }},
		git: &git.MockClient{
			CurrentBranchFunc:            func() (string, error) { return "main", nil },
			IsBranchSyncedWithRemoteFunc: func(string) error { return nil },
		},
		pr:       &promptPRRecorder{},
		output:   &promptOutputRecorder{},
		remote:   &promptWorkflowClient{ctx: ctx, argoClient: argoClient},
		NoFollow: true,
	}

	require.NoError(t, cmd.RunRemoteFromClone("do work", "update/fix-auth", "main", "update"))

	assert.Equal(t, "target-ns", submittedCtx.Namespace, "the target namespace wins over the configured namespace")
	assert.Empty(t, submittedCtx.Name, "the configured context does not apply in a pod")

	var data map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(submittedYAML), &data))

	metadata := data["metadata"].(map[string]interface{})
	labels := metadata["labels"].(map[string]interface{})
	assert.Equal(t, "clone-team", labels["team"])

	spec := data["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	assert.Equal(t, "registry.example/clone:clone-v1", container["image"])
	require.NotEmpty(t, container["resources"])

	hasEnv := func(env []interface{}, name, value string) bool {
		for _, entry := range env {
			envVar, ok := entry.(map[string]interface{})
			if ok && envVar["name"] == name && envVar["value"] == value {
				return true
			}
		}
		return false
	}
	hasMount := func(mounts interface{}, name, path string) bool {
		entries, ok := mounts.([]interface{})
		if !ok {
			return false
		}
		for _, entry := range entries {
			mount, ok := entry.(map[string]interface{})
			if ok && mount["name"] == name && mount["mountPath"] == path {
				return true
			}
		}
		return false
	}
	hasVolume := func(volumes interface{}, name string) bool {
		entries, ok := volumes.([]interface{})
		if !ok {
			return false
		}
		for _, entry := range entries {
			volume, ok := entry.(map[string]interface{})
			if ok && volume["name"] == name {
				return true
			}
		}
		return false
	}

	env := container["env"].([]interface{})
	assert.True(t, hasEnv(env, "CLONE_VAR", "clone-value"), "the clone environment reaches the workflow")

	assert.True(t, hasMount(container["volumeMounts"], "clone-config", "/configmaps/clone-config"),
		"the clone ConfigMap reaches the workflow")
	assert.True(t, hasMount(container["volumeMounts"], "clone-secret", "/secrets/clone-secret"),
		"the clone Secret reaches the workflow")
	assert.True(t, hasVolume(tmpl["volumes"], "clone-config"), "the clone ConfigMap volume reaches the workflow")
	assert.True(t, hasVolume(tmpl["volumes"], "clone-secret"), "the clone Secret volume reaches the workflow")
}

// TestPromptWorkflowRunnerAdapterWithoutNamespaceRunsInPod covers a prompt with
// no target namespace keeping the existing in-pod flow.
func TestPromptWorkflowRunnerAdapterWithoutNamespaceRunsInPod(t *testing.T) {
	remote := &promptRemoteRecorder{}
	aiClient := &promptAIRecorder{}
	promptCmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git: &git.MockClient{
			HeadCommitFunc: func() (string, error) { return "tip", nil },
		},
		services: &promptServicesRecorder{},
		ai:       aiClient,
		pr:       &promptPRRecorder{},
		output:   &promptOutputRecorder{},
		remote:   remote,
	}
	adapter := &promptWorkflowRunnerAdapter{
		ctx:          execcontext.NewContext(),
		newPromptCmd: func(*execcontext.Context) *PromptCmd { return promptCmd },
	}

	err := adapter.Run(orchestrationPrompt.WorkflowPromptFlags{
		Branch:     "my-feature",
		BaseBranch: "main",
		Prompt:     "do work",
		NoServices: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, aiClient.passCalls, "the prompt runs in this pod")
	assert.Zero(t, remote.submitCalls, "no workflow is submitted without a target namespace")
}
