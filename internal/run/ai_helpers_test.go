package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestGeneratePRSummaryNoProject(t *testing.T) {
	ctx := testutil.NewContext()

	proj := &project.Project{Name: "Test Project"}

	_, err := GeneratePRSummary(ctx, proj, "", "main", "")
	assert.Error(t, err, "GeneratePRSummary should fail with nonexistent project file")
}

func TestGenerateReviewPRBodyNoProject(t *testing.T) {
	ctx := testutil.NewContext()

	proj := &project.Project{Name: "Review Project"}

	_, err := GenerateReviewPRBody(ctx, proj, []string{})
	assert.Error(t, err, "GenerateReviewPRBody should fail with nonexistent project file")
}
