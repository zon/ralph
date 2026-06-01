package github

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
)

func TestGitHubClientNew(t *testing.T) {
	ctx := context.NewContext()
	client := NewClient(ctx, "main")
	require.NotNil(t, client)
	var _ orchestrationRun.GitHubClient = client
}
