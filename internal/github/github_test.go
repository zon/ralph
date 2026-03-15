package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePR(t *testing.T) {
	_, _ = CreatePR("Test PR", "Test body", "main", "feature-branch")
}

func TestMakeRepo(t *testing.T) {
	repo := MakeRepo("zon", "ralph")
	assert.Equal(t, "zon", repo.Owner)
	assert.Equal(t, "ralph", repo.Name)
}

func TestCloneURL(t *testing.T) {
	assert.Equal(t, "https://github.com/zon/ralph.git", CloneURL("zon", "ralph"))
}
