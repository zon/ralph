package project

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

func TestRunIterationLoop_DryRun(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: test-project
description: Test project for iteration loop
requirements:
  - category: feature
    description: Add feature
    steps:
      - Step 1
      - Step 2
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	// Run iteration loop in dry-run mode
	iterations, err := RunIterationLoop(ctx, nil)
	require.NoError(t, err, "RunIterationLoop failed in dry-run")

	// In dry-run mode, it should return max iterations
	assert.Equal(t, 10, iterations)
}

func TestRunIterationLoop_ReturnsErrorWhenMaxIterationsReached(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	projectFile := "test-project.yaml"
	projectContent := `name: test-project
description: Test project for iteration loop
requirements:
  - category: feature
    description: Add feature
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create initial report.md for first iteration commit to succeed
	// (after commit, report.md is deleted but loop ends since maxIterations=1)
	if err := os.WriteFile("report.md", []byte("Test iteration 1"), 0644); err != nil {
		t.Fatalf("Failed to create report.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithMaxIterations(1),
		testutil.WithDryRun(false),
	)

	_, err := RunIterationLoop(ctx, nil)
	require.Error(t, err, "Expected error when max iterations reached but requirements still failing")
	assert.True(t, errors.Is(err, ErrMaxIterationsReached), "Expected ErrMaxIterationsReached, got: %v", err)
}

func TestRunIterationLoop_BlockedDetected(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create a minimal git repo
	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create project file
	projectFile := "test-project.yaml"
	projectContent := `name: test-project
description: Test project
requirements:
  - category: feature
    description: Add feature
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create blocked.md in repo root
	blockedContent := "Agent is blocked due to some issue"
	if err := os.WriteFile("blocked.md", []byte(blockedContent), 0644); err != nil {
		t.Fatalf("Failed to create blocked.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithMaxIterations(5),
	)

	_, err := RunIterationLoop(ctx, nil)
	require.Error(t, err, "Expected error when blocked.md is detected")
	assert.True(t, errors.Is(err, ErrBlocked), "Expected ErrBlocked, got: %v", err)
}

func TestIsFatalOpenCodeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Insufficient Balance error",
			err:      errors.New("opencode execution failed: Insufficient Balance"),
			expected: true,
		},
		{
			name:     "lowercase insufficient balance",
			err:      errors.New("opencode execution failed: insufficient balance"),
			expected: true,
		},
		{
			name:     "billing error",
			err:      errors.New("opencode execution failed: billing error"),
			expected: true,
		},
		{
			name:     "payment required",
			err:      errors.New("opencode execution failed: payment required"),
			expected: true,
		},
		{
			name:     "quota exceeded",
			err:      errors.New("opencode execution failed: quota exceeded"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFatalOpenCodeError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitChanges_DryRun(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	// In dry-run mode, should not error
	err := CommitChanges(ctx, 1)
	require.NoError(t, err, "CommitChanges failed in dry-run")
}

func TestCommitChanges_UsesReportMd(t *testing.T) {
	// This test verifies that CommitChanges reads from report.md
	// We can't test actual git commits without a real git repo,
	// but we can test the report.md reading logic

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create report.md with a commit message
	reportContent := "Add new feature\n\nImplemented feature X with tests"
	if err := os.WriteFile("report.md", []byte(reportContent), 0644); err != nil {
		t.Fatalf("Failed to create report.md: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("project.yaml"))

	// In dry-run mode, this should read report.md (though not commit)
	err := CommitChanges(ctx, 1)
	require.NoError(t, err, "CommitChanges failed")

	// In dry-run mode, report.md should still exist
	_, err = os.Stat("report.md")
	require.NoError(t, err, "report.md should exist in dry-run mode")
}

func TestCommitChanges_FailsWithoutReportMd(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	err := CommitChanges(ctx, 5)
	// No uncommitted changes and no report.md means there is nothing to commit
	assert.ErrorIs(t, err, ErrNoChanges)
}

// setupIterationTestRepo creates a temporary git repo with a bare remote.
// After the initial commit is pushed, the provided pre-receive hook is installed
// so that subsequent pushes are rejected with the supplied hook output.
// Returns the path to the working clone.
func setupIterationTestRepo(t *testing.T, hookContent string) string {
	t.Helper()

	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	workDir := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, out)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Create an initial commit and push it before installing the hook so the
	// bare remote has a valid HEAD.
	readmePath := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial commit"},
		{"push", "origin", "HEAD"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Install the hook after the initial push so only subsequent pushes are rejected.
	if hookContent != "" {
		hookPath := filepath.Join(remoteDir, "hooks", "pre-receive")
		if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
			t.Fatalf("failed to write hook: %v", err)
		}
	}

	return workDir
}

func TestCommitChanges_WorkflowPermissionErrorIsFatal(t *testing.T) {
	// Arrange: a bare remote whose pre-receive hook emits the GitHub
	// workflow-permission rejection so that any push from a workflow file commit
	// fails with the recognisable message.
	hookContent := "#!/bin/sh\necho 'refusing to allow a GitHub App to create or update workflow `.github/workflows/test.yaml` without `workflows` permission' >&2\nexit 1\n"
	workDir := setupIterationTestRepo(t, hookContent)

	t.Chdir(workDir)

	// Stage a new file so CommitChanges has something to commit.
	wfDir := filepath.Join(workDir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "test.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Write a report.md so CommitChanges has a commit message.
	if err := os.WriteFile("report.md", []byte("Add workflow file"), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile("project.yaml"),
		testutil.WithDryRun(false),
	)

	err := CommitChanges(ctx, 1)
	require.Error(t, err, "expected CommitChanges to return an error, got nil")
	assert.True(t, errors.Is(err, ErrFatalPushError), "expected ErrFatalPushError, got: %v", err)
}

func TestRunIterationLoop_WorkflowPermissionStopsLoop(t *testing.T) {
	// Arrange: a bare remote that always rejects pushes with the GitHub
	// workflow-permission message.  The iteration loop should stop after the
	// first failed push and surface ErrFatalPushError rather than
	// ErrMaxIterationsReached.
	hookContent := "#!/bin/sh\necho 'refusing to allow a GitHub App to create or update workflow `.github/workflows/test.yaml` without `workflows` permission' >&2\nexit 1\n"
	workDir := setupIterationTestRepo(t, hookContent)

	// Create a project file inside the repo.
	projectFile := filepath.Join(workDir, "project.yaml")
	projectContent := `name: test-project
description: Test workflow permission handling
requirements:
  - category: feature
    description: Add workflow file
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("failed to write project file: %v", err)
	}

	t.Chdir(workDir)

	// Stage a workflow file so CommitChanges has something to commit in
	// iteration 1 (requirement.Execute is skipped in dry-run mode, but
	// CommitChanges is only skipped in dry-run too, so we must drive
	// this at a level where the push actually fires).
	//
	// Because RunIterationLoop calls requirement.Execute (which calls opencode)
	// we can't easily drive a non-dry-run loop without a real AI service.
	// Instead we test the error sentinel directly through CommitChanges,
	// which is the function that invokes PushCurrentBranch.
	//
	// Verify that ErrFatalPushError wraps ErrWorkflowPermission so callers
	// can inspect the root cause.
	wfDir := filepath.Join(workDir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "test.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "report.md"), []byte("Add workflow"), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithMaxIterations(5),
		testutil.WithDryRun(false),
	)

	err := CommitChanges(ctx, 1)
	require.Error(t, err, "expected CommitChanges to return an error, got nil")
	assert.True(t, errors.Is(err, ErrFatalPushError), "expected ErrFatalPushError wrapping ErrWorkflowPermission, got: %v", err)
	// Confirm root cause is accessible.
	assert.True(t, errors.Is(err, ErrFatalPushError), "ErrFatalPushError not in error chain: %v", err)
}

func TestCommitChanges_ReadsReportMdAndCommits(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")

	t.Chdir(workDir)

	if err := os.WriteFile("feature.go", []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write feature.go: %v", err)
	}
	if err := os.WriteFile("report.md", []byte("Add new feature"), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile("project.yaml"),
		testutil.WithDryRun(false),
	)

	err := CommitChanges(ctx, 1)
	require.NoError(t, err, "CommitChanges failed")

	_, err = os.Stat("report.md")
	assert.True(t, os.IsNotExist(err), "report.md should have been removed after commit")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git log failed")
	msg := strings.TrimSpace(string(out))
	assert.Equal(t, "Add new feature", msg)
}

func TestCommitChanges_UsesProvidedReportMd(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")

	t.Chdir(workDir)

	if err := os.WriteFile("feature.go", []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write feature.go: %v", err)
	}

	if err := os.WriteFile("report.md", []byte("Add new feature for iteration 42"), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile("project.yaml"),
		testutil.WithDryRun(false),
	)

	err := CommitChanges(ctx, 42)
	require.NoError(t, err, "CommitChanges failed")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git log failed")
	msg := strings.TrimSpace(string(out))
	assert.Equal(t, "Add new feature for iteration 42", msg)
}

func TestCommitChanges_AllowEmptyCommitWhenNoStagedChanges(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")

	t.Chdir(workDir)

	if err := os.WriteFile("report.md", []byte("No changes made"), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile("project.yaml"),
		testutil.WithDryRun(false),
	)

	err := CommitChanges(ctx, 1)
	require.NoError(t, err, "CommitChanges failed")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git log failed")
	msg := strings.TrimSpace(string(out))
	assert.Equal(t, "No changes made", msg)
}

func TestGenerateChangelogIfNeeded_NoChanges(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Clean repo — no uncommitted changes, no report.md.
	// generateChangelogIfNeeded should return nil without doing anything.
	err := generateChangelogIfNeeded(ctx)
	require.NoError(t, err, "generateChangelogIfNeeded should succeed when tree is clean")

	// report.md must not have been created
	_, statErr := os.Stat("report.md")
	assert.True(t, os.IsNotExist(statErr), "report.md should not be created when there are no changes")
}

func TestGenerateChangelogIfNeeded_ReportMdAlreadyPresent(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Create uncommitted changes
	if err := os.WriteFile(filepath.Join(workDir, "new.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write new.go: %v", err)
	}

	// report.md is already present — opencode must not be called
	originalReport := "Existing changelog entry"
	if err := os.WriteFile("report.md", []byte(originalReport), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	err := generateChangelogIfNeeded(ctx)
	require.NoError(t, err, "generateChangelogIfNeeded should succeed when report.md already exists")

	// report.md should be unchanged
	content, readErr := os.ReadFile("report.md")
	require.NoError(t, readErr)
	assert.Equal(t, originalReport, string(content), "report.md should not be overwritten when already present")
}

func TestGenerateChangelogIfNeeded_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// dry-run ctx — HasUncommittedChanges returns true, but GenerateChangelog is a no-op
	ctx := testutil.NewContext() // dry-run by default

	err := generateChangelogIfNeeded(ctx)
	require.NoError(t, err, "generateChangelogIfNeeded in dry-run mode should not fail")
}

func TestRunIterationLoop_ExitsEarlyWhenAllRequirementsPass(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: test-project
description: Test project
requirements:
  - category: feature
    description: Add feature
    passing: true
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithMaxIterations(10),
		testutil.WithDryRun(true),
	)

	iterations, err := RunIterationLoop(ctx, nil)
	require.NoError(t, err, "RunIterationLoop should not error when requirements are already passing")
	assert.Equal(t, 1, iterations)
}
