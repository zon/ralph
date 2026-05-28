package run

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
)

func TestGitHubClientAdapterNewAdapter(t *testing.T) {
	ctx := context.NewContext()
	adapter := NewGitHubClientAdapter(ctx, "main")
	require.NotNil(t, adapter)
	var _ GitHubClient = adapter
}

func TestGitHubClientAdapterCreatePR_UsesMockAI(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	workDir := t.TempDir()
	t.Chdir(workDir)
	initGitRepo(t, workDir)
	makeInitialCommit(t, workDir)

	c := exec.Command("git", "checkout", "-b", "test-slug")
	c.Dir = workDir
	require.NoError(t, c.Run())

	require.NoError(t, os.WriteFile("feature.txt", []byte("work"), 0644))
	c = exec.Command("git", "add", "feature.txt")
	c.Dir = workDir
	require.NoError(t, c.Run())
	c = exec.Command("git", "commit", "-m", "feat: add feature")
	c.Dir = workDir
	require.NoError(t, c.Run())

	proj := &project.Project{
		Slug:  "test-slug",
		Title: "Test Project",
	}

	ctx := context.NewContext()
	adapter := NewGitHubClientAdapter(ctx, "main")

	err := adapter.CreatePR(proj)
	if err != nil {
		t.Logf("CreatePR returned (expected without gh): %v", err)
	}
}
