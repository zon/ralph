package ai

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/testutil"
)

func TestBuildFixServicePrompt(t *testing.T) {
	ctx := testutil.NewContext()
	svcErr := fmt.Errorf("failed to start service test-service: connection refused")

	svc := config.Service{
		Name:    "test-service",
		Command: "myapp",
		Args:    []string{"--port", "8080"},
		Port:    8080,
	}
	prompt, err := BuildFixServicePrompt(ctx, svc, svcErr)
	require.NoError(t, err, "BuildFixServicePrompt failed")

	assert.True(t, strings.Contains(prompt, "Service Startup Failed"), "Prompt does not contain service failure header")
	assert.True(t, strings.Contains(prompt, "failed to start service test-service"), "Prompt does not contain service failure details")
	assert.True(t, strings.Contains(prompt, "test-service"), "Prompt does not contain service name")
	assert.True(t, strings.Contains(prompt, "myapp --port 8080"), "Prompt does not contain start command")
	assert.True(t, strings.Contains(prompt, "port 8080"), "Prompt does not contain health check port")
	assert.False(t, strings.Contains(prompt, "## Project Requirements"), "Service fix prompt should not contain project requirements")
	assert.False(t, strings.Contains(prompt, "**Recent Git History:**"), "Service fix prompt should not contain git history")
	assert.True(t, strings.Contains(prompt, "report.md"), "Service fix prompt should contain report.md instruction")
}

func TestBuildFixServicePrompt_NoPort(t *testing.T) {
	ctx := testutil.NewContext()
	svcErr := fmt.Errorf("failed to start service worker: exit status 1")

	svc := config.Service{
		Name:    "worker",
		Command: "worker",
		Args:    []string{"--config", "worker.yaml"},
	}
	prompt, err := BuildFixServicePrompt(ctx, svc, svcErr)
	require.NoError(t, err, "BuildFixServicePrompt failed")

	assert.True(t, strings.Contains(prompt, "worker --config worker.yaml"), "Prompt does not contain start command")
	assert.False(t, strings.Contains(prompt, "Health check"), "Prompt should not contain health check when no port configured")
}

func TestBuildFixServicePrompt_ReturnsError(t *testing.T) {
	ctx := testutil.NewContext()
	svcErr := fmt.Errorf("service failed")

	svc := config.Service{
		Name:    "test",
		Command: "test",
	}

	_, err := BuildFixServicePrompt(ctx, svc, svcErr)
	require.NoError(t, err, "BuildFixServicePrompt should not error with valid inputs")
}

func TestBuildFixServicePrompt_ErrorNil(t *testing.T) {
	ctx := testutil.NewContext()
	svcErr := errors.New("test error")

	svc := config.Service{
		Name:    "test",
		Command: "test",
	}

	_, err := BuildFixServicePrompt(ctx, svc, svcErr)
	require.NoError(t, err, "BuildFixServicePrompt should not error with valid inputs")
}

func TestBuildDevelopPrompt(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "",
		ProjectContent:      "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false",
		SelectedRequirement: "- slug: feature-x\n  description: Feature X\n  passing: false",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "# Development Agent")
	assert.Contains(t, prompt, "## Context")
	assert.Contains(t, prompt, "**Selected Requirement:**")
	assert.Contains(t, prompt, "## Instructions")
	assert.Contains(t, prompt, "Feature X")
	assert.Contains(t, prompt, "/path/to/project.yaml")
	assert.Contains(t, prompt, "Implement the selected requirement")
	assert.NotContains(t, prompt, "**Recent Git History:**")
	assert.NotContains(t, prompt, "**System Notes:**")
}

func TestBuildDevelopPrompt_WithNotes(t *testing.T) {
	data := DevelopPromptData{
		Notes:               []string{"Note 1", "Note 2"},
		CommitLog:           "",
		ProjectContent:      "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false",
		SelectedRequirement: "- slug: feature-x\n  description: Feature X\n  passing: false",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "**System Notes:**")
	assert.Contains(t, prompt, "Note 1")
	assert.Contains(t, prompt, "Note 2")
}

func TestBuildDevelopPrompt_WithCommitLog(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "abc123 Feature A\ndef456 Feature B",
		ProjectContent:      "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false",
		SelectedRequirement: "- slug: feature-x\n  description: Feature X\n  passing: false",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "**Recent Git History:**")
	assert.Contains(t, prompt, "abc123 Feature A")
	assert.Contains(t, prompt, "def456 Feature B")
}

func TestBuildDevelopPrompt_WithServices(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "",
		ProjectContent:      "slug: test-project\ntitle: Test Project",
		SelectedRequirement: "- slug: feature-x\n  description: Feature X",
		ProjectFilePath:     "/path/to/project.yaml",
		Services: []config.Service{
			{Name: "api", Command: "api-server"},
			{Name: "worker", Command: "worker"},
		},
		Instructions: config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "**Services**")
	assert.Contains(t, prompt, "api.log")
	assert.Contains(t, prompt, "worker.log")
}

func TestBuildDevelopPrompt_WithCustomInstructions(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "",
		ProjectContent:      "slug: test-project\ntitle: Test Project",
		SelectedRequirement: "- slug: feature-x\n  description: Feature X",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        "Custom instructions: Do something special",
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "Custom instructions: Do something special")
}

func TestBuildDevelopPrompt_ErrorOnInvalidTemplate(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "",
		ProjectContent:      "content",
		SelectedRequirement: "requirement",
		ProjectFilePath:     "/path",
		Services:            nil,
		Instructions:        "{{.InvalidField}}",
	}

	_, err := BuildDevelopPrompt(data)
	require.Error(t, err, "BuildDevelopPrompt should error on invalid template")
	assert.Contains(t, err.Error(), "failed to execute template")
}

func TestBuildPickPrompt(t *testing.T) {
	data := PickPromptData{
		Notes:          nil,
		CommitLog:      "",
		ProjectContent: "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false\n  - slug: bug-y\n    description: Bug Y\n    passing: false",
		PickedReqPath:  "/path/to/picked-requirement.yaml",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.Contains(t, prompt, "# Requirement Picker Agent")
	assert.Contains(t, prompt, "## Context")
	assert.Contains(t, prompt, "**Project Requirements:**")
	assert.Contains(t, prompt, "## Instructions")
	assert.Contains(t, prompt, "Test Project")
	assert.Contains(t, prompt, "Feature X")
	assert.Contains(t, prompt, "Bug Y")
	assert.Contains(t, prompt, "/path/to/picked-requirement.yaml")
	assert.NotContains(t, prompt, "**Recent Git History:**")
	assert.NotContains(t, prompt, "**System Notes:**")
}

func TestBuildPickPrompt_WithNotes(t *testing.T) {
	data := PickPromptData{
		Notes:          []string{"Pick this one"},
		CommitLog:      "",
		ProjectContent: "slug: test-project\ntitle: Test Project",
		PickedReqPath:  "/path/to/picked-requirement.yaml",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.Contains(t, prompt, "**System Notes:**")
	assert.Contains(t, prompt, "Pick this one")
}

func TestBuildPickPrompt_WithCommitLog(t *testing.T) {
	data := PickPromptData{
		Notes:          nil,
		CommitLog:      "abc123 Initial commit\ndef456 Add feature",
		ProjectContent: "slug: test-project\ntitle: Test Project",
		PickedReqPath:  "/path/to/picked-requirement.yaml",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.Contains(t, prompt, "**Recent Git History:**")
	assert.Contains(t, prompt, "abc123 Initial commit")
}

func TestExecuteTemplate(t *testing.T) {
	type testData struct {
		Name string
		Age  int
	}

	result, err := executeTemplate("Name: {{.Name}}, Age: {{.Age}}", testData{Name: "Alice", Age: 30})
	require.NoError(t, err, "executeTemplate failed")
	assert.Equal(t, "Name: Alice, Age: 30", result)
}

func TestExecuteTemplate_ParseError(t *testing.T) {
	_, err := executeTemplate("{{.Invalid", nil)
	require.Error(t, err, "executeTemplate should error on parse failure")
	assert.Contains(t, err.Error(), "failed to parse template")
}

func TestExecuteTemplate_ExecuteError(t *testing.T) {
	type testData struct {
		Name string
	}

	_, err := executeTemplate("{{.Age}}", testData{Name: "Bob"})
	require.Error(t, err, "executeTemplate should error on execute failure")
	assert.Contains(t, err.Error(), "failed to execute template")
}

func TestBuildDevelopPrompt_EmptyNotes(t *testing.T) {
	data := DevelopPromptData{
		Notes:               []string{},
		CommitLog:           "",
		ProjectContent:      "slug: test",
		SelectedRequirement: "- slug: x\n  description: X",
		ProjectFilePath:     "/path",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.NotContains(t, prompt, "**System Notes:**")
}

func TestBuildPickPrompt_EmptyNotes(t *testing.T) {
	data := PickPromptData{
		Notes:          []string{},
		CommitLog:      "",
		ProjectContent: "slug: test",
		PickedReqPath:  "/path",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.NotContains(t, prompt, "**System Notes:**")
}

func TestBuildPRSummaryPrompt(t *testing.T) {
	prompt, err := BuildPRSummaryPrompt(
		"Test Project",
		"✅ Complete",
		"main",
		"abc123: Initial commit\ndef456: Add feature\n",
		"/tmp/pr-summary.txt",
	)

	require.NoError(t, err, "BuildPRSummaryPrompt failed")
	assert.NotEmpty(t, prompt, "PR summary prompt should not be empty")
	assert.Contains(t, prompt, "Test Project", "prompt should include project description")
	assert.Contains(t, prompt, "✅ Complete", "prompt should include project status")
	assert.Contains(t, prompt, "main..HEAD", "prompt should reference base branch")
	assert.Contains(t, prompt, "abc123: Initial commit", "prompt should include commit log")
	assert.Contains(t, prompt, "/tmp/pr-summary.txt", "prompt should include output file path")
}

func TestBuildPRSummaryPrompt_AbsolutePath(t *testing.T) {
	prompt, err := BuildPRSummaryPrompt(
		"My Project",
		"status",
		"develop",
		"commit log",
		"relative/path.txt",
	)

	require.NoError(t, err, "BuildPRSummaryPrompt failed")
	absPath, _ := filepath.Abs("relative/path.txt")
	assert.Contains(t, prompt, absPath, "prompt should contain absolute path")
}

func TestBuildChangelogPrompt(t *testing.T) {
	prompt, err := BuildChangelogPrompt("/tmp/report.md")

	require.NoError(t, err, "BuildChangelogPrompt failed")
	assert.NotEmpty(t, prompt, "changelog prompt should not be empty")
	assert.Contains(t, prompt, "report.md", "prompt should reference report.md")
	assert.Contains(t, prompt, "git diff", "prompt should instruct inspecting git diff")
}

func TestBuildChangelogPrompt_AbsolutePath(t *testing.T) {
	prompt, err := BuildChangelogPrompt("changelog.txt")

	require.NoError(t, err, "BuildChangelogPrompt failed")
	absPath, _ := filepath.Abs("changelog.txt")
	assert.Contains(t, prompt, absPath, "prompt should contain absolute path")
}

func TestBuildReviewPRBodyPrompt(t *testing.T) {
	prompt, err := BuildReviewPRBodyPrompt(
		"review-2026-03-22",
		"Code review for authentication",
		[]string{"- **security**: JWT validation (✅ Passing)", "- **style**: naming conventions (❌ Not passing)"},
		"/tmp/pr-body.txt",
	)

	require.NoError(t, err, "BuildReviewPRBodyPrompt failed")
	assert.NotEmpty(t, prompt, "PR body prompt should not be empty")
	assert.Contains(t, prompt, "review-2026-03-22", "prompt should include review name")
	assert.Contains(t, prompt, "Code review for authentication", "prompt should include description")
	assert.Contains(t, prompt, "JWT validation", "prompt should include requirement details")
	assert.Contains(t, prompt, "/tmp/pr-body.txt", "prompt should include output file path")
}

func TestBuildReviewPRBodyPrompt_NoDescription(t *testing.T) {
	prompt, err := BuildReviewPRBodyPrompt(
		"review-2026-03-22",
		"",
		[]string{"- **security**: JWT validation (✅ Passing)"},
		"/tmp/pr-body.txt",
	)

	require.NoError(t, err, "BuildReviewPRBodyPrompt failed")
	assert.NotEmpty(t, prompt, "PR body prompt should not be empty")
	assert.Contains(t, prompt, "review-2026-03-22", "prompt should include review name")
	assert.NotContains(t, prompt, "Description:", "prompt should not include empty description")
}

func TestBuildReviewPRBodyPrompt_AbsolutePath(t *testing.T) {
	prompt, err := BuildReviewPRBodyPrompt(
		"review",
		"description",
		[]string{"req1", "req2"},
		"relative/path.txt",
	)

	require.NoError(t, err, "BuildReviewPRBodyPrompt failed")
	absPath, _ := filepath.Abs("relative/path.txt")
	assert.Contains(t, prompt, absPath, "prompt should contain absolute path")
}

func TestBuildArchitecturePrompt(t *testing.T) {
	prompt, err := BuildArchitecturePrompt("/tmp/architecture.yaml")

	require.NoError(t, err, "BuildArchitecturePrompt failed")
	assert.NotEmpty(t, prompt, "architecture prompt should not be empty")
	assert.Contains(t, prompt, "architecture.yaml", "prompt should reference architecture.yaml")
	assert.Contains(t, prompt, "software architect", "prompt should describe architect role")
	assert.Contains(t, prompt, "domain function", "prompt should define domain functions")
	assert.Contains(t, prompt, "Major Feature", "prompt should define major features")
	assert.Contains(t, prompt, "cmd/", "prompt should mention cmd/ for app discovery")
	assert.Contains(t, prompt, "internal/", "prompt should mention internal/ for module discovery")
	assert.Contains(t, prompt, "/tmp/architecture.yaml", "prompt should include output file path")
}

func TestBuildArchitecturePrompt_AbsolutePath(t *testing.T) {
	prompt, err := BuildArchitecturePrompt("architecture.yaml")

	require.NoError(t, err, "BuildArchitecturePrompt failed")
	absPath, _ := filepath.Abs("architecture.yaml")
	assert.Contains(t, prompt, absPath, "prompt should contain absolute path")
}

func TestBuildArchitectureFixPrompt(t *testing.T) {
	errors := []string{"app 'ralph' is missing description", "module 'internal/ai' must have type domain or implementation"}
	prompt, err := BuildArchitectureFixPrompt("/tmp/architecture.yaml", errors)

	require.NoError(t, err, "BuildArchitectureFixPrompt failed")
	assert.NotEmpty(t, prompt, "architecture fix prompt should not be empty")
	assert.Contains(t, prompt, "/tmp/architecture.yaml", "prompt should include output file path")
	assert.Contains(t, prompt, "validation errors", "prompt should mention validation errors")
	assert.Contains(t, prompt, "app 'ralph' is missing description", "prompt should include first error")
	assert.Contains(t, prompt, "module 'internal/ai' must have type domain or implementation", "prompt should include second error")
	assert.Contains(t, prompt, "## Errors", "prompt should include Errors section")
}

func TestBuildArchitectureFixPrompt_AbsolutePath(t *testing.T) {
	errors := []string{"test error"}
	prompt, err := BuildArchitectureFixPrompt("architecture.yaml", errors)

	require.NoError(t, err, "BuildArchitectureFixPrompt failed")
	absPath, _ := filepath.Abs("architecture.yaml")
	assert.Contains(t, prompt, absPath, "prompt should contain absolute path")
}

func TestBuildArchitectureFixPrompt_EmptyErrors(t *testing.T) {
	prompt, err := BuildArchitectureFixPrompt("/tmp/architecture.yaml", []string{})

	require.NoError(t, err, "BuildArchitectureFixPrompt failed")
	assert.NotEmpty(t, prompt, "architecture fix prompt should not be empty")
	assert.Contains(t, prompt, "/tmp/architecture.yaml", "prompt should include output file path")
}
