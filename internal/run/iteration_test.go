package run

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestRunIterationLoop_AllPassingExitsAfterOneIteration(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	// With all requirements already passing, the loop should exit after 1 iteration
	// without invoking AI (when using mock AI). CommitChanges returns ErrNoChanges
	// (no report.md, no uncommitted changes), which the loop treats as a non-error.
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	projectFile := "test-project.yaml"
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

	// Commit the project file so it's not an uncommitted change during the loop
	for _, args := range [][]string{
		{"add", projectFile},
		{"commit", "-m", "add project file"},
		{"push", "origin", "HEAD"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithMaxIterations(10),
	)

	proj, err := project.LoadProject(projectFile)
	require.NoError(t, err)

	iterations, err := RunIterationLoop(ctx, nil, proj)
	require.NoError(t, err, "RunIterationLoop should not error when requirements are already passing")
	assert.Equal(t, 1, iterations)
}

func TestRunIterationLoop_ReturnsErrorWhenMaxIterationsReached(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

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
	)

	proj, err := project.LoadProject(projectFile)
	require.NoError(t, err)

	_, err = RunIterationLoop(ctx, nil, proj)
	require.Error(t, err, "Expected error when max iterations reached but requirements still failing")
	assert.True(t, errors.Is(err, ErrMaxIterationsReached), "Expected ErrMaxIterationsReached, got: %v", err)
}

func TestRunIterationLoop_BlockedDetected(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

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

	proj, err := project.LoadProject(projectFile)
	require.NoError(t, err)

	_, err = RunIterationLoop(ctx, nil, proj)
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

func TestGenerateChangelogIfNeeded_NoChanges(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	ctx := testutil.NewContext()

	err := generateChangelogIfNeeded(ctx)
	require.NoError(t, err, "generateChangelogIfNeeded should succeed when tree is clean")

	_, statErr := os.Stat("report.md")
	assert.True(t, os.IsNotExist(statErr), "report.md should not be created when there are no changes")
}

func TestGenerateChangelogIfNeeded_ReportMdAlreadyPresent(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	ctx := testutil.NewContext()

	if err := os.WriteFile(filepath.Join(workDir, "new.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write new.go: %v", err)
	}

	originalReport := "Existing changelog entry"
	if err := os.WriteFile("report.md", []byte(originalReport), 0644); err != nil {
		t.Fatalf("failed to write report.md: %v", err)
	}

	err := generateChangelogIfNeeded(ctx)
	require.NoError(t, err, "generateChangelogIfNeeded should succeed when report.md already exists")

	content, readErr := os.ReadFile("report.md")
	require.NoError(t, readErr)
	assert.Equal(t, originalReport, string(content), "report.md should not be overwritten when already present")
}

func TestGenerateChangelogIfNeeded_WithUncommittedChanges(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ctx := testutil.NewContext()

	err := generateChangelogIfNeeded(ctx)
	require.NoError(t, err, "generateChangelogIfNeeded should not fail with mock AI")
}

func TestRunIterationLoop_ExitsEarlyWhenAllRequirementsPass(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	projectFile := "test-project.yaml"
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

	for _, args := range [][]string{
		{"add", projectFile},
		{"commit", "-m", "add project file"},
		{"push", "origin", "HEAD"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithMaxIterations(10),
	)

	proj, err := project.LoadProject(projectFile)
	require.NoError(t, err)

	iterations, err := RunIterationLoop(ctx, nil, proj)
	require.NoError(t, err, "RunIterationLoop should not error when requirements are already passing")
	assert.Equal(t, 1, iterations)
}

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

	if hookContent != "" {
		hookPath := filepath.Join(remoteDir, "hooks", "pre-receive")
		if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
			t.Fatalf("failed to write hook: %v", err)
		}
	}

	ralphDir := filepath.Join(workDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("failed to create .ralph directory: %v", err)
	}
	repoConfig, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load repo config: %v", err)
	}
	configContent := "model: " + repoConfig.Model + "\n"
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create .ralph/config.yaml: %v", err)
	}

	for _, args := range [][]string{
		{"add", ".ralph"},
		{"commit", "-m", "add ralph config"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	return workDir
}
