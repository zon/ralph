package run

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveBaseBranchExplicitBaseFlag(t *testing.T) {
	result := ResolveBaseBranch("develop", "feature-x", "my-project", "main")
	require.Equal(t, "develop", result)
}

func TestResolveBaseBranchCurrentDiffersFromProject(t *testing.T) {
	result := ResolveBaseBranch("", "feature-x", "my-project", "main")
	require.Equal(t, "feature-x", result)
}

func TestResolveBaseBranchAlreadyOnProjectBranch(t *testing.T) {
	result := ResolveBaseBranch("", "my-project", "my-project", "main")
	require.Equal(t, "main", result)
}
