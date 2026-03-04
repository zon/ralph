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

// TestRun_RemoteWorkflow submits a real Argo Workflow to the test repository
// using a pre-completed project file. Because all requirements are already
// passing: true, the iteration loop exits after the first iteration without
// invoking the AI, making the test deterministic and fast.
//
// What this exercises end-to-end:
//   - workflow YAML generation (GenerateWorkflow)
//   - argo submit (real Argo CLI call)
//   - container startup and git clone
//   - ralph --local execution inside the container (git checkout, iteration loop)
//   - git commit + push from inside the container
//   - gh pr create from inside the container
//   - PR visible on GitHub
func TestRun_RemoteWorkflow(t *testing.T) {
	cfg := resolveConfig(t)

	// Write the pre-completed project file to a temp dir so we have an absolute path.
	// The project file name becomes the branch name (ralph/e2e-noop-run).
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "e2e-noop-run.yaml")
	require.NoError(t, os.WriteFile(projectFile, noopProjectYAML(), 0644))

	ctx := &context.Context{
		ProjectFile: projectFile,
		Repo:        cfg.Repo,
		Branch:      cfg.Branch,
		DebugBranch: cfg.DebugBranch,
		DryRun:      false,
		Local:       false, // remote: submit an Argo Workflow
		Verbose:     true,
		NoNotify:    true,
		NoServices:  true,
	}

	// Generate the workflow — this calls GenerateWorkflow which uses ctx.Repo to
	// skip local git commands, giving us a hermetic test even without a real local repo.
	t.Log("Generating Argo Workflow...")
	projectName := "e2e-noop-run"
	cloneBranch := cfg.Branch
	projectBranch := "e2e-noop-run"

	wf, err := workflow.GenerateWorkflowWithGitInfo(
		ctx,
		projectName,
		"https://github.com/"+cfg.Repo+".git",
		cloneBranch,
		projectBranch,
		"test-data/e2e-noop-run.yaml",
		false, // dryRun inside container
		true,  // verbose inside container
	)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	// Render and log the workflow YAML for debugging.
	yamlBytes, err := wf.Render()
	require.NoError(t, err)
	t.Logf("Workflow YAML:\n%s", yamlBytes)

	// Submit.
	t.Log("Submitting workflow to Argo...")
	workflowName, err := wf.Submit(cfg.Namespace)
	require.NoError(t, err, "workflow submission failed")
	t.Logf("Submitted workflow: %s", workflowName)

	// Register cleanup: delete the test branch and close the PR regardless of outcome.
	branch := projectBranch
	t.Cleanup(func() {
		cleanupBranchAndPR(t, cfg.Repo, branch)
	})

	// Poll until the workflow succeeds or the timeout is reached.
	t.Logf("Waiting for workflow %s to complete (timeout: %s)...", workflowName, cfg.Timeout)
	err = pollWorkflowCompletion(t, workflowName, cfg.Namespace, cfg.Timeout)
	require.NoError(t, err, "workflow did not complete successfully")

	// Verify the PR was created on GitHub.
	t.Log("Verifying PR was created on GitHub...")
	prURL, err := findPR(cfg.Repo, branch)
	require.NoError(t, err, "could not find PR for branch %s", branch)
	assert.NotEmpty(t, prURL, "expected a PR URL")
	t.Logf("PR created: %s", prURL)
}

// TestRun_DryRunWorkflow submits a real Argo Workflow but the ralph command
// inside the container runs with --dry-run, so no git writes or PRs are created.
// This validates the workflow YAML, container startup, and ralph binary invocation
// without any side effects on the test repository.
func TestRun_DryRunWorkflow(t *testing.T) {
	cfg := resolveConfig(t)

	ctx := &context.Context{
		Repo:        cfg.Repo,
		Branch:      cfg.Branch,
		DebugBranch: cfg.DebugBranch,
		DryRun:      false,
		Local:       false,
		Verbose:     true,
		NoNotify:    true,
		NoServices:  true,
	}

	projectName := "e2e-dryrun"
	projectBranch := "e2e-dryrun"

	wf, err := workflow.GenerateWorkflowWithGitInfo(
		ctx,
		projectName,
		"https://github.com/"+cfg.Repo+".git",
		cfg.Branch,
		projectBranch,
		"test-data/e2e-noop-run.yaml",
		true, // dryRun inside container — no side effects
		true,
	)
	require.NoError(t, err)

	workflowName, err := wf.Submit(cfg.Namespace)
	require.NoError(t, err, "workflow submission failed")
	t.Logf("Submitted dry-run workflow: %s", workflowName)

	err = pollWorkflowCompletion(t, workflowName, cfg.Namespace, cfg.Timeout)
	require.NoError(t, err, "dry-run workflow did not complete successfully")
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

	ctx := &context.Context{
		ProjectFile:   projectFile,
		MaxIterations: 3,
		DryRun:        true, // suppress push and PR creation; real git reads still work
		Local:         true,
		Verbose:       true,
		NoNotify:      true,
		NoServices:    true,
	}

	// Change cwd so config.LoadConfig() finds the (absent) .ralph/config.yaml —
	// it falls back to defaults, which is fine.
	originalDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(repoDir))
	t.Cleanup(func() { os.Chdir(originalDir) })

	err := project.Execute(ctx, nil)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// noopProjectYAML returns a project YAML where all requirements are already
// passing: true. The iteration loop exits after the first check without calling
// the AI, making E2E tests fast and deterministic.
func noopProjectYAML() []byte {
	return []byte(`name: e2e-noop-run
description: E2E test project — all requirements pre-completed
requirements:
  - id: noop
    category: e2e
    description: No-op requirement (always passing)
    passing: true
`)
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
