//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/cmd"
)

// TestMerge_CleanStatus verifies that ghMerge handles the "clean status" error
// from GitHub: when a PR is immediately mergeable (all checks passed, no queue),
// enablePullRequestAutoMerge is rejected and ghMerge must fall back to a direct
// merge without --auto.
//
// Exercises end-to-end:
//   - create a base branch and a head branch on ralph-mock via the GitHub API
//   - open a PR from head → base
//   - call MergeCmd.Run (--local) which calls ghMerge
//   - assert the PR is merged and both branches are deleted
//
// The PR is immediately mergeable because ralph-mock has no required status
// checks or review rules on ad-hoc branches, so GitHub returns "clean status"
// when --auto is attempted.
func TestMerge_CleanStatus(t *testing.T) {
	cfg := resolveConfig(t)

	baseBranch := "e2e-merge-base"
	headBranch := "e2e-merge-head"

	// Start base branch from main.
	mainSHA := getRemoteBranchSHA(t, cfg.Repo, cfg.Branch)
	createRemoteBranch(t, cfg.Repo, baseBranch, mainSHA)
	t.Cleanup(func() { cleanupBranchAndPR(t, cfg.Repo, headBranch) })
	t.Cleanup(func() { deleteBranch(t, cfg.Repo, baseBranch) })

	// Create head branch with one commit ahead of base so the PR has a diff.
	headSHA := createCommitOnBranch(t, cfg.Repo, baseBranch, headBranch)
	t.Logf("Created head branch %s at %s", headBranch, headSHA[:8])

	// Open a PR from head → base.
	prNumber := openPR(t, cfg.Repo, headBranch, baseBranch, "e2e: merge clean-status test")
	t.Logf("Opened PR #%d (%s → %s)", prNumber, headBranch, baseBranch)

	// Run MergeCmd --local, which calls ghMerge under the hood.
	mergeCmd := &cmd.MergeCmd{
		Branch: headBranch,
		PR:     fmt.Sprintf("%d", prNumber),
		Local:  true,
		Repo:   cfg.Repo,
	}
	err := mergeCmd.Run()
	require.NoError(t, err, "MergeCmd.Run failed — ghMerge clean-status fallback may not have worked")

	// Confirm the PR is now merged.
	state := getPRState(t, cfg.Repo, prNumber)
	assert.Equal(t, "MERGED", state, "PR should be merged after ghMerge")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createCommitOnBranch creates headBranch from baseBranch and adds a single
// file commit to it via the GitHub contents API. Returns the new commit SHA.
func createCommitOnBranch(t *testing.T, repo, baseBranch, headBranch string) string {
	t.Helper()

	// Create head branch pointing at base.
	baseSHA := getRemoteBranchSHA(t, repo, baseBranch)
	createRemoteBranch(t, repo, headBranch, baseSHA)

	// Add a file via the contents API so the branch has a commit ahead of base.
	path := fmt.Sprintf("e2e-merge-test-%s.txt", headBranch)
	out, err := exec.Command(
		"gh", "api",
		"--method", "PUT",
		fmt.Sprintf("repos/%s/contents/%s", repo, path),
		"--field", fmt.Sprintf("message=e2e: add file on %s", headBranch),
		"--field", "content=ZTJlIG1lcmdlIHRlc3QK", // base64("e2e merge test\n")
		"--field", fmt.Sprintf("branch=%s", headBranch),
	).CombinedOutput()
	require.NoError(t, err, "failed to create commit on %s: %s", headBranch, out)

	return getRemoteBranchSHA(t, repo, headBranch)
}

// openPR opens a pull request from head → base in repo and returns the PR number.
func openPR(t *testing.T, repo, head, base, title string) int {
	t.Helper()
	out, err := exec.Command(
		"gh", "pr", "create",
		"--repo", repo,
		"--head", head,
		"--base", base,
		"--title", title,
		"--body", "Created by TestMerge_CleanStatus e2e test.",
	).CombinedOutput()
	require.NoError(t, err, "gh pr create failed: %s", out)

	// Extract the PR number from the URL printed by gh pr create.
	url := strings.TrimSpace(string(out))
	parts := strings.Split(url, "/")
	var prNumber int
	_, parseErr := fmt.Sscanf(parts[len(parts)-1], "%d", &prNumber)
	require.NoError(t, parseErr, "could not parse PR number from gh output: %s", out)
	return prNumber
}

// getPRState returns the state of a PR ("OPEN", "CLOSED", or "MERGED").
func getPRState(t *testing.T, repo string, prNumber int) string {
	t.Helper()
	out, err := exec.Command(
		"gh", "pr", "view", fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--json", "state",
		"--jq", ".state",
	).Output()
	require.NoError(t, err, "gh pr view failed")
	return strings.TrimSpace(string(out))
}

// deleteBranch deletes a remote branch via the GitHub API (best-effort).
func deleteBranch(t *testing.T, repo, branch string) {
	t.Helper()
	exec.Command(
		"gh", "api",
		"--method", "DELETE",
		"-H", "Accept: application/vnd.github+json",
		fmt.Sprintf("/repos/%s/git/refs/heads/%s", repo, branch),
	).Run()
}
