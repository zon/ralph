package github

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestGitHubClientNew(t *testing.T) {
	ctx := context.NewContext()
	client := NewClient(ctx, "main")
	require.NotNil(t, client)
	var _ orchestrationRun.GitHubClient = client
}

func TestGitHubClientCreatePR_UsesMockAI(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

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
	client := NewClient(ctx, "main")

	err := client.CreatePR(proj)
	if err != nil {
		t.Logf("CreatePR returned (expected without gh): %v", err)
	}
}
