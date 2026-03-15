package github

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/testutil"
)

func TestCreatePR(t *testing.T) {
	ctx := testutil.NewContext()
	_, _ = CreatePR(ctx, "Test PR", "Test body", "main", "feature-branch")
}

func TestMakeRepo(t *testing.T) {
	repo := MakeRepo("zon", "ralph")
	assert.Equal(t, "zon", repo.Owner)
	assert.Equal(t, "ralph", repo.Name)
}

func TestCloneURL(t *testing.T) {
	assert.Equal(t, "https://github.com/zon/ralph.git", CloneURL("zon", "ralph"))
}
