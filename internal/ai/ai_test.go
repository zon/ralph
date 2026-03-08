package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

func TestRunAgentDryRun(t *testing.T) {
	ctx := testutil.NewContext()

	err := RunAgent(ctx, "test prompt")
	require.NoError(t, err, "RunAgent in dry-run mode should not fail")
}

func TestGeneratePRSummaryDryRun(t *testing.T) {
	ctx := testutil.NewContext()

	summary, err := GeneratePRSummary(ctx, "test.yaml", 3)
	require.NoError(t, err, "GeneratePRSummary in dry-run mode should not fail")
	assert.Equal(t, "dry-run-pr-summary", summary, "Expected dry-run-pr-summary")
}

func TestGeneratePRSummaryNoProject(t *testing.T) {
	ctx := testutil.NewContext(testutil.WithDryRun(false))

	_, err := GeneratePRSummary(ctx, "nonexistent.yaml", 1)
	assert.Error(t, err, "GeneratePRSummary should fail with nonexistent project file")
}
