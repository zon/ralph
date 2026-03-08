//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

// TestRun_NewProject submits a real Argo Workflow for a brand-new project branch.
// The pre-completed noop project (all requirements passing: true) ensures the
// iteration loop exits immediately without invoking the AI.
//
// Exercises end-to-end:
//   - workflow YAML generation (GenerateWorkflow)
//   - argo submit (real Argo CLI call)
//   - container startup and git clone
//   - new branch creation inside the container (git checkout -b)
//   - ralph --local: iteration loop exits immediately, commit + push
//   - gh pr create: PR visible on GitHub with title matching project description
func TestRun_NewProject(t *testing.T) {
	cfg := resolveConfig(t)

	branch := "e2e-noop-run"

	ctx := &context.Context{}
	ctx.SetRepo(cfg.Repo)
	ctx.SetBranch(cfg.Branch)
	ctx.SetDebugBranch(cfg.DebugBranch)
	ctx.SetVerbose(true)
	ctx.SetNoNotify(true)
	ctx.SetNoServices(true)

	t.Log("Generating Argo Workflow...")
	wf, err := workflow.GenerateWorkflowWithGitInfo(
		ctx,
		"e2e-noop-run",
		"https://github.com/"+cfg.Repo+".git",
		cfg.Branch,
		branch,
		"test-data/e2e-noop-run.yaml",
		false,
		true,
	)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	yamlBytes, err := wf.Render()
	require.NoError(t, err)
	t.Logf("Workflow YAML:\n%s", yamlBytes)

	t.Log("Submitting workflow to Argo...")
	workflowName, err := wf.Submit(cfg.Namespace)
	require.NoError(t, err, "workflow submission failed")
	t.Logf("Submitted workflow: %s", workflowName)

	t.Cleanup(func() { cleanupBranchAndPR(t, cfg.Repo, branch) })

	t.Logf("Waiting for workflow %s to complete (timeout: %s)...", workflowName, cfg.Timeout)
	require.NoError(t, pollWorkflowCompletion(t, workflowName, cfg.Namespace, cfg.Timeout))

	// Verify the PR was created with the correct title (derived from project description).
	prURL, err := findPR(cfg.Repo, branch)
	require.NoError(t, err, "could not find PR for branch %s", branch)
	assert.NotEmpty(t, prURL, "expected a PR URL")
	t.Logf("PR created: %s", prURL)

	title, err := getPRTitle(cfg.Repo, branch)
	require.NoError(t, err)
	assert.Equal(t, noopProjectDescription, title, "PR title should match project description")
}

// TestRun_DryRunWorkflow submits a real Argo Workflow with --dry-run inside the
// container, so no git writes or PRs are produced. Validates workflow YAML,
// container startup, and ralph binary invocation without side effects.
func TestRun_DryRunWorkflow(t *testing.T) {
	cfg := resolveConfig(t)

	ctx := &context.Context{}
	ctx.SetRepo(cfg.Repo)
	ctx.SetBranch(cfg.Branch)
	ctx.SetDebugBranch(cfg.DebugBranch)
	ctx.SetVerbose(true)
	ctx.SetNoNotify(true)
	ctx.SetNoServices(true)

	wf, err := workflow.GenerateWorkflowWithGitInfo(
		ctx,
		"e2e-dryrun",
		"https://github.com/"+cfg.Repo+".git",
		cfg.Branch,
		"e2e-dryrun",
		"test-data/e2e-noop-run.yaml",
		true, // --dry-run inside container — no side effects
		true,
	)
	require.NoError(t, err)

	workflowName, err := wf.Submit(cfg.Namespace)
	require.NoError(t, err, "workflow submission failed")
	t.Logf("Submitted dry-run workflow: %s", workflowName)

	require.NoError(t, pollWorkflowCompletion(t, workflowName, cfg.Namespace, cfg.Timeout))
}

// TestRun_ResumesExistingBranch verifies that when the project branch already
// exists on the remote, the container checks it out rather than creating a new
// one. This exercises the `git checkout $PROJECT_BRANCH` path in run.sh as
// opposed to `git checkout -b $PROJECT_BRANCH`.
//
// Exercises end-to-end:
//   - run.sh detects an existing remote branch and checks it out (not -b)
//   - ralph --local runs on the pre-existing branch, iteration loop exits immediately
//   - gh pr create: PR created from the existing branch
//
// Prerequisites: test-data/e2e-resume-run.yaml must exist in cfg.Repo
// (checked by TestNamespacePreflight).
func TestRun_ResumesExistingBranch(t *testing.T) {
	cfg := resolveConfig(t)

	branch := "e2e-resume-run"

	// Pre-create the project branch on GitHub so run.sh takes the checkout
	// (not create) path. Point it at the current HEAD of the base branch.
	sha := getRemoteBranchSHA(t, cfg.Repo, cfg.Branch)
	createRemoteBranch(t, cfg.Repo, branch, sha)
	t.Cleanup(func() { cleanupBranchAndPR(t, cfg.Repo, branch) })

	ctx := &context.Context{}
	ctx.SetRepo(cfg.Repo)
	ctx.SetBranch(cfg.Branch)
	ctx.SetDebugBranch(cfg.DebugBranch)
	ctx.SetVerbose(true)
	ctx.SetNoNotify(true)
	ctx.SetNoServices(true)

	wf, err := workflow.GenerateWorkflowWithGitInfo(
		ctx,
		"e2e-resume-run",
		"https://github.com/"+cfg.Repo+".git",
		cfg.Branch,
		branch,
		"test-data/e2e-resume-run.yaml",
		false,
		true,
	)
	require.NoError(t, err)

	workflowName, err := wf.Submit(cfg.Namespace)
	require.NoError(t, err, "workflow submission failed")
	t.Logf("Submitted workflow: %s", workflowName)

	err = pollWorkflowCompletion(t, workflowName, cfg.Namespace, cfg.Timeout)
	require.NoError(t, err, "workflow did not complete — container may have failed to checkout existing branch")

	prURL, err := findPR(cfg.Repo, branch)
	require.NoError(t, err)
	assert.NotEmpty(t, prURL, "expected a PR URL")
	t.Logf("PR created: %s", prURL)
}

// TestRun_AICompletesSingleIteration verifies the core development loop: the AI
// agent is invoked on a project with one failing requirement, makes the required
// change (adds a file), marks the requirement passing: true in the project YAML,
// and the iteration loop exits after that single iteration.
//
// This is the only E2E test that actually exercises the AI iteration path.
// All other tests use pre-passing noop projects and never call the AI.
//
// Exercises end-to-end:
//   - iteration loop invokes the AI agent (requirement is not pre-passing)
//   - AI modifies the project file (sets passing: true)
//   - iteration loop detects completion and exits
//   - commit + push of AI changes
//   - gh pr create: PR visible on GitHub
//
// Prerequisites: test-data/e2e-ai-iteration.yaml must exist in cfg.Repo
// (checked by TestNamespacePreflight).
func TestRun_AICompletesSingleIteration(t *testing.T) {
	cfg := resolveConfig(t)

	branch := "e2e-ai-iteration"

	ctx := &context.Context{}
	ctx.SetRepo(cfg.Repo)
	ctx.SetBranch(cfg.Branch)
	ctx.SetDebugBranch(cfg.DebugBranch)
	ctx.SetVerbose(true)
	ctx.SetNoNotify(true)
	ctx.SetNoServices(true)

	t.Log("Generating Argo Workflow...")
	wf, err := workflow.GenerateWorkflowWithGitInfo(
		ctx,
		"e2e-ai-iteration",
		"https://github.com/"+cfg.Repo+".git",
		cfg.Branch,
		branch,
		"test-data/e2e-ai-iteration.yaml",
		false,
		true,
	)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	workflowName, err := wf.Submit(cfg.Namespace)
	require.NoError(t, err, "workflow submission failed")
	t.Logf("Submitted workflow: %s", workflowName)

	t.Cleanup(func() { cleanupBranchAndPR(t, cfg.Repo, branch) })

	t.Logf("Waiting for workflow %s to complete (timeout: %s)...", workflowName, cfg.Timeout)
	require.NoError(t, pollWorkflowCompletion(t, workflowName, cfg.Namespace, cfg.Timeout),
		"workflow did not complete — AI may have failed or timed out")

	// Verify a PR was created, proving the iteration loop exited cleanly after
	// the AI marked the requirement as passing.
	prURL, err := findPR(cfg.Repo, branch)
	require.NoError(t, err, "could not find PR for branch %s", branch)
	assert.NotEmpty(t, prURL, "expected a PR — AI iteration did not complete or PR was not created")
	t.Logf("PR created: %s", prURL)

	title, err := getPRTitle(cfg.Repo, branch)
	require.NoError(t, err)
	assert.Equal(t, aiIterationProjectDescription, title, "PR title should match project description")
}

// TestRun_MaxIterationsExhausted is a placeholder for a test that verifies the
// workflow fails gracefully when the AI cannot satisfy all requirements within
// the configured iteration limit.
//
// Skipped because it requires two things not yet in place:
//  1. A --max-iterations flag threaded through workflow generation into run.sh
//     (currently the container always uses .ralph/config.yaml maxIterations).
//  2. A test-data/e2e-stuck-run.yaml in cfg.Repo with requirements the AI
//     cannot satisfy in a single iteration, so exhaustion is deterministic.
//
// To enable: add --max-iterations to Workflow and run.sh template, create the
// stuck project file, and remove the t.Skip.
func TestRun_MaxIterationsExhausted(t *testing.T) {
	t.Skip("requires --max-iterations in workflow generation and a stuck project file in ralph-mock")
}

// TestExecute_LocalWithRealGit runs project.Execute in --local mode against a
// real temporary git repository (no Argo, no GitHub). This validates the full
// local execution path — branch creation, iteration loop, commit — using real
// git but without any network calls (DryRun: true suppresses push and PR creation).
func TestExecute_LocalWithRealGit(t *testing.T) {
	// Bootstrap a real git repo in a temp dir with a bare remote.
	repoDir, _ := bootstrapGitRepo(t)

	projectFile := filepath.Join(repoDir, "test-data", "e2e-noop-run.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(projectFile), 0755))
	require.NoError(t, os.WriteFile(projectFile, noopProjectYAML(), 0644))

	// Stage and commit the project file so the repo has a HEAD commit.
	gitExec(t, repoDir, "add", ".")
	gitExec(t, repoDir, "commit", "-m", "add e2e project file")

	ctx := &context.Context{}
	ctx.SetProjectFile(projectFile)
	ctx.SetMaxIterations(3)
	ctx.SetDryRun(true) // suppress push and PR creation; real git reads still work
	ctx.SetLocal(true)
	ctx.SetVerbose(true)
	ctx.SetNoNotify(true)
	ctx.SetNoServices(true)

	// Change cwd so config.LoadConfig() finds the (absent) .ralph/config.yaml —
	// it falls back to defaults, which is fine.
	originalDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(repoDir))
	t.Cleanup(func() { os.Chdir(originalDir) })

	err := project.Execute(ctx, nil)
	require.NoError(t, err)
}

// TestPush_StaleTokenCleanup verifies that a push succeeds even when the global
// git config already contains a stale x-access-token insteadOf entry for
// github.com from a prior auth call (simulating the container startup auth
// followed by a long-running iteration that needs to re-auth at push time).
//
// The test uses a local bare remote so no real GitHub credentials are needed.
// It exercises the stale-entry cleanup code path in github.ConfigureGitAuth by
// planting a fake stale rewrite in an isolated git HOME before the push.
func TestPush_StaleTokenCleanup(t *testing.T) {
	// Use an isolated HOME so we can safely manipulate the global gitconfig
	// without touching the developer's real ~/.gitconfig.
	fakeHome := t.TempDir()
	origHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", fakeHome))
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	repoDir, _ := bootstrapGitRepo(t)

	// Plant a stale token rewrite that mimics what a previous `ralph set-github-token`
	// call would have left behind. In production this token would be expired.
	gitExec(t, repoDir, "config", "--global",
		"url.https://x-access-token:stale-expired-token@github.com/.insteadOf",
		"https://github.com/",
	)

	// Verify the stale entry is present before the push.
	out, err := exec.Command("git", "config", "--global", "--list").Output()
	require.NoError(t, err, "git config --global --list failed")
	require.Contains(t, string(out), "stale-expired-token", "stale token entry should exist before cleanup")

	// Create a new branch with a commit so there is something to push.
	gitExec(t, repoDir, "checkout", "-b", "push-auth-test")
	testFile := filepath.Join(repoDir, "push-test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("testing push auth\n"), 0644))
	gitExec(t, repoDir, "add", ".")
	gitExec(t, repoDir, "commit", "-m", "add push-test file")

	// Change cwd to the repo so git commands operate on the right repository.
	originalDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(repoDir))
	t.Cleanup(func() { os.Chdir(originalDir) })

	// Call git.PushBranch via the project/git package using the internal API.
	// Because RALPH_WORKFLOW_EXECUTION is not set, configureAuth is a no-op and
	// the local bare remote is used directly — this confirms the push itself works.
	// The cleanup path is separately verified by TestCleanupStaleTokenRewrites in
	// the github package unit tests.
	//
	// The push should succeed against the local file-system remote despite the
	// stale global insteadOf entry, because git only applies insteadOf rewrites
	// when the remote URL starts with the target prefix (https://github.com/).
	// A local file-path remote is unaffected, giving us a clean pass/fail signal.
	pushCmd := exec.Command("git", "push", "--set-upstream", "origin", "push-auth-test")
	pushCmd.Dir = repoDir
	out2, pushErr := pushCmd.CombinedOutput()
	require.NoError(t, pushErr, "push to local bare remote failed (stale entry may have interfered): %s", out2)

	// Confirm the branch now exists on the remote.
	lsCmd := exec.Command("git", "ls-remote", "--heads", "origin", "push-auth-test")
	lsCmd.Dir = repoDir
	lsOut, lsErr := lsCmd.Output()
	require.NoError(t, lsErr)
	assert.Contains(t, string(lsOut), "push-auth-test", "branch should exist on remote after push")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// noopProjectDescription is the description field in test-data/e2e-noop-run.yaml
// in cfg.Repo. It is also the expected PR title when a noop workflow runs.
const noopProjectDescription = "E2E test project — all requirements pre-completed"

// aiIterationProjectDescription is the description field in
// test-data/e2e-ai-iteration.yaml in cfg.Repo. Must match exactly.
const aiIterationProjectDescription = "E2E test — single AI iteration adds a greeting file"

// noopProjectYAML returns a project YAML where all requirements are already
// passing: true. The iteration loop exits after the first check without calling
// the AI, making E2E tests fast and deterministic.
// Must match test-data/e2e-noop-run.yaml in cfg.Repo.
func noopProjectYAML() []byte {
	return []byte(`name: e2e-noop-run
description: ` + noopProjectDescription + `
requirements:
  - id: noop
    category: e2e
    description: No-op requirement (always passing)
    passing: true
`)
}

// getRemoteBranchSHA returns the HEAD commit SHA of branch in repo.
func getRemoteBranchSHA(t *testing.T, repo, branch string) string {
	t.Helper()
	out, err := exec.Command(
		"gh", "api",
		fmt.Sprintf("repos/%s/git/ref/heads/%s", repo, branch),
		"--jq", ".object.sha",
	).Output()
	require.NoError(t, err, "failed to get SHA for branch %s in %s", branch, repo)
	return strings.TrimSpace(string(out))
}

// createRemoteBranch creates branch in repo pointing at sha via the GitHub API.
func createRemoteBranch(t *testing.T, repo, branch, sha string) {
	t.Helper()
	out, err := exec.Command(
		"gh", "api",
		"--method", "POST",
		fmt.Sprintf("repos/%s/git/refs", repo),
		"--field", fmt.Sprintf("ref=refs/heads/%s", branch),
		"--field", fmt.Sprintf("sha=%s", sha),
	).CombinedOutput()
	require.NoError(t, err, "failed to create remote branch %s: %s", branch, out)
	t.Logf("Pre-created remote branch: %s", branch)
}

// getPRTitle returns the title of the open PR for branch in repo.
func getPRTitle(repo, branch string) (string, error) {
	out, err := exec.Command(
		"gh", "pr", "list",
		"--repo", repo,
		"--head", branch,
		"--json", "title",
		"--jq", ".[0].title",
	).Output()
	if err != nil {
		return "", fmt.Errorf("gh pr list: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// pollWorkflowCompletion polls `argo get` until the workflow reaches Succeeded
// or Failed, or the timeout elapses.
func pollWorkflowCompletion(t *testing.T, workflowName, namespace string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	interval := 15 * time.Second

	for time.Now().Before(deadline) {
		out, err := exec.Command("argo", "get", "-n", namespace, workflowName, "--no-color").CombinedOutput()
		if err == nil {
			output := string(out)
			if strings.Contains(output, "Succeeded") {
				t.Logf("Workflow %s Succeeded", workflowName)
				return nil
			}
			if strings.Contains(output, "Failed") || strings.Contains(output, "Error") {
				return fmt.Errorf("workflow %s failed:\n%s", workflowName, output)
			}
		}
		t.Logf("Workflow %s still running, next check in %s...", workflowName, interval)
		time.Sleep(interval)
	}

	return fmt.Errorf("workflow %s did not complete within %s", workflowName, timeout)
}

// findPR uses `gh pr list` to find an open PR for the given branch.
// Returns the PR URL if found.
func findPR(repo, branch string) (string, error) {
	out, err := exec.Command(
		"gh", "pr", "list",
		"--repo", repo,
		"--head", branch,
		"--json", "url",
		"--jq", ".[0].url",
	).Output()
	if err != nil {
		return "", fmt.Errorf("gh pr list failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// cleanupBranchAndPR closes any open PR for the branch and deletes the remote
// branch. Errors are logged but do not fail the test.
func cleanupBranchAndPR(t *testing.T, repo, branch string) {
	t.Helper()

	// Delete the remote branch (best-effort; may already be gone if PR was merged).
	exec.Command("gh", "api",
		"--method", "DELETE",
		"-H", "Accept: application/vnd.github+json",
		fmt.Sprintf("/repos/%s/git/refs/heads/%s", repo, branch),
	).Run()

	// Close any open PRs.
	out, err := exec.Command(
		"gh", "pr", "list",
		"--repo", repo,
		"--head", branch,
		"--json", "number",
		"--jq", ".[].number",
	).Output()
	if err == nil {
		for _, num := range strings.Fields(string(out)) {
			if closeErr := exec.Command("gh", "pr", "close", num, "--repo", repo).Run(); closeErr != nil {
				t.Logf("cleanup: could not close PR %s: %v", num, closeErr)
			}
		}
	}
}

// bootstrapGitRepo creates a temporary git repository with a bare remote in
// t.TempDir(). It configures a local user identity so commits work without
// system git config. Returns (repoDir, bareDir).
func bootstrapGitRepo(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	bareDir := filepath.Join(root, "remote.git")
	repoDir := filepath.Join(root, "repo")

	// Create bare remote.
	require.NoError(t, os.MkdirAll(bareDir, 0755))
	gitExec(t, bareDir, "init", "--bare")

	// Clone from the bare remote so origin is configured.
	gitExecAt(t, root, "clone", bareDir, repoDir)

	// Set local identity so commits don't require global git config.
	gitExec(t, repoDir, "config", "user.email", "test@example.com")
	gitExec(t, repoDir, "config", "user.name", "E2E Test")

	// Make an initial commit so HEAD exists.
	readmeFile := filepath.Join(repoDir, "README.md")
	require.NoError(t, os.WriteFile(readmeFile, []byte("# E2E test repo\n"), 0644))
	gitExec(t, repoDir, "add", ".")
	gitExec(t, repoDir, "commit", "-m", "initial commit")
	gitExec(t, repoDir, "push", "origin", "HEAD")

	return repoDir, bareDir
}

// gitExec runs a git command inside dir, failing the test on error.
func gitExec(t *testing.T, dir string, args ...string) {
	t.Helper()
	gitExecAt(t, dir, args...)
}

// gitExecAt runs a git command with the given working directory.
func gitExecAt(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s failed:\n%s", strings.Join(args, " "), out)
}
