package github

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func skipIfGHNotAvailable(t *testing.T) {
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("gh CLI not available")
	}
}

func TestCreatePullRequest_UsesTitleAsPRTitle(t *testing.T) {
	skipIfGHNotAvailable(t)
	t.Setenv("RALPH_MOCK_GH", "true")

	proj := &project.Project{
		Slug:  "test-project",
		Title: "This is a detailed title",
	}
	ctx := testutil.NewContext()

	prURL, err := CreatePullRequest(ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
	assert.Contains(t, prURL, "github.com")
}

func TestCreatePullRequest_UsesSlugWhenTitleEmpty(t *testing.T) {
	skipIfGHNotAvailable(t)
	t.Setenv("RALPH_MOCK_GH", "true")

	proj := &project.Project{
		Slug:  "my-project",
		Title: "",
	}
	ctx := testutil.NewContext()

	prURL, err := CreatePullRequest(ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func TestCreatePullRequest_UsesSlugWhenTitleMissing(t *testing.T) {
	skipIfGHNotAvailable(t)
	t.Setenv("RALPH_MOCK_GH", "true")

	proj := &project.Project{
		Slug: "fallback-project",
	}
	ctx := testutil.NewContext()

	prURL, err := CreatePullRequest(ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}
