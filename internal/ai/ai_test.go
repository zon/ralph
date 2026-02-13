package ai

import (
	"testing"

	"github.com/zon/ralph/internal/context"
)

func TestRunAgentDryRun(t *testing.T) {
	ctx := context.NewContext(true, false, true, false)

	err := RunAgent(ctx, "test prompt")
	if err != nil {
		t.Errorf("RunAgent in dry-run mode should not fail: %v", err)
	}
}

// TestRunAgentNoOpenCode is removed since RunAgent now delegates to OpenCode CLI
// OpenCode manages its own configuration and will fail appropriately if not configured

func TestGeneratePRSummaryDryRun(t *testing.T) {
	ctx := context.NewContext(true, false, true, false)

	summary, err := GeneratePRSummary(ctx, "test.yaml", 3)
	if err != nil {
		t.Errorf("GeneratePRSummary in dry-run mode should not fail: %v", err)
	}

	if summary != "dry-run-pr-summary" {
		t.Errorf("Expected dry-run-pr-summary, got: %s", summary)
	}
}

func TestGeneratePRSummaryNoProject(t *testing.T) {
	ctx := context.NewContext(false, false, true, false)

	_, err := GeneratePRSummary(ctx, "nonexistent.yaml", 1)
	if err == nil {
		t.Error("GeneratePRSummary should fail with nonexistent project file")
	}
}
