package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveBaseBranch_ExplicitBaseFlag(t *testing.T) {
	base := resolveBaseBranch("develop", "feature-x", "my-project", "main")
	assert.Equal(t, "develop", base)
}

func TestResolveBaseBranch_CurrentBranchDiffersFromProject(t *testing.T) {
	base := resolveBaseBranch("", "feature-x", "my-project", "main")
	assert.Equal(t, "feature-x", base)
}

func TestResolveBaseBranch_AlreadyOnProjectBranch(t *testing.T) {
	base := resolveBaseBranch("", "my-project", "my-project", "main")
	assert.Equal(t, "main", base)
}
