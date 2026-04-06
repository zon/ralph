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

func TestCreatePullRequest_UsesDescriptionAsTitle(t *testing.T) {
	skipIfGHNotAvailable(t)
	t.Setenv("RALPH_MOCK_GH", "true")

	proj := &project.Project{
		Name:        "Test Project",
		Description: "This is a detailed description",
	}
	ctx := testutil.NewContext()

	prURL, err := CreatePullRequest(ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
	assert.Contains(t, prURL, "github.com")
}

func TestCreatePullRequest_UsesNameWhenDescriptionEmpty(t *testing.T) {
	skipIfGHNotAvailable(t)
	t.Setenv("RALPH_MOCK_GH", "true")

	proj := &project.Project{
		Name:        "My Project",
		Description: "",
	}
	ctx := testutil.NewContext()

	prURL, err := CreatePullRequest(ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func TestCreatePullRequest_UsesNameWhenDescriptionMissing(t *testing.T) {
	skipIfGHNotAvailable(t)
	t.Setenv("RALPH_MOCK_GH", "true")

	proj := &project.Project{
		Name: "Fallback Project",
	}
	ctx := testutil.NewContext()

	prURL, err := CreatePullRequest(ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}
