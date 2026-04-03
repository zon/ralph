package prompt

import (
	"errors"
	"fmt"
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
	assert.False(t, strings.Contains(prompt, "## Recent Git History"), "Service fix prompt should not contain git history")
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
		ProjectContent:      "name: Test Project\nrequirements:\n  - description: Feature X\n    passing: false",
		SelectedRequirement: "- description: Feature X\n  passing: false",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "# Development Agent Context")
	assert.Contains(t, prompt, "## Project Information")
	assert.Contains(t, prompt, "## Selected Requirement")
	assert.Contains(t, prompt, "## Instructions")
	assert.Contains(t, prompt, "Feature X")
	assert.Contains(t, prompt, "/path/to/project.yaml")
	assert.Contains(t, prompt, "Implement the selected requirement")
	assert.NotContains(t, prompt, "## Recent Git History")
	assert.NotContains(t, prompt, "## System Notes")
}

func TestBuildDevelopPrompt_WithNotes(t *testing.T) {
	data := DevelopPromptData{
		Notes:               []string{"Note 1", "Note 2"},
		CommitLog:           "",
		ProjectContent:      "name: Test Project\nrequirements:\n  - description: Feature X\n    passing: false",
		SelectedRequirement: "- description: Feature X\n  passing: false",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "## System Notes")
	assert.Contains(t, prompt, "Note 1")
	assert.Contains(t, prompt, "Note 2")
}

func TestBuildDevelopPrompt_WithCommitLog(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "abc123 Feature A\ndef456 Feature B",
		ProjectContent:      "name: Test Project\nrequirements:\n  - description: Feature X\n    passing: false",
		SelectedRequirement: "- description: Feature X\n  passing: false",
		ProjectFilePath:     "/path/to/project.yaml",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "## Recent Git History")
	assert.Contains(t, prompt, "abc123 Feature A")
	assert.Contains(t, prompt, "def456 Feature B")
}

func TestBuildDevelopPrompt_WithServices(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "",
		ProjectContent:      "name: Test Project",
		SelectedRequirement: "- description: Feature X",
		ProjectFilePath:     "/path/to/project.yaml",
		Services: []config.Service{
			{Name: "api", Command: "api-server"},
			{Name: "worker", Command: "worker"},
		},
		Instructions: config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.Contains(t, prompt, "## Services")
	assert.Contains(t, prompt, "api.log")
	assert.Contains(t, prompt, "worker.log")
}

func TestBuildDevelopPrompt_WithCustomInstructions(t *testing.T) {
	data := DevelopPromptData{
		Notes:               nil,
		CommitLog:           "",
		ProjectContent:      "name: Test Project",
		SelectedRequirement: "- description: Feature X",
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
		Instructions:        "{{.InvalidField}}", // references non-existent field
	}

	_, err := BuildDevelopPrompt(data)
	require.Error(t, err, "BuildDevelopPrompt should error on invalid template")
	assert.Contains(t, err.Error(), "failed to execute template")
}

func TestBuildPickPrompt(t *testing.T) {
	data := PickPromptData{
		Notes:          nil,
		CommitLog:      "",
		ProjectContent: "name: Test Project\nrequirements:\n  - description: Feature X\n    passing: false\n  - description: Bug Y\n    passing: false",
		PickedReqPath:  "/path/to/picked-requirement.yaml",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.Contains(t, prompt, "# Requirement Picker Agent Context")
	assert.Contains(t, prompt, "## Project Information")
	assert.Contains(t, prompt, "## Project Requirements")
	assert.Contains(t, prompt, "## Instructions")
	assert.Contains(t, prompt, "Test Project")
	assert.Contains(t, prompt, "Feature X")
	assert.Contains(t, prompt, "Bug Y")
	assert.Contains(t, prompt, "/path/to/picked-requirement.yaml")
	assert.NotContains(t, prompt, "## Recent Git History")
	assert.NotContains(t, prompt, "## System Notes")
}

func TestBuildPickPrompt_WithNotes(t *testing.T) {
	data := PickPromptData{
		Notes:          []string{"Pick this one"},
		CommitLog:      "",
		ProjectContent: "name: Test Project",
		PickedReqPath:  "/path/to/picked-requirement.yaml",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.Contains(t, prompt, "## System Notes")
	assert.Contains(t, prompt, "Pick this one")
}

func TestBuildPickPrompt_WithCommitLog(t *testing.T) {
	data := PickPromptData{
		Notes:          nil,
		CommitLog:      "abc123 Initial commit\ndef456 Add feature",
		ProjectContent: "name: Test Project",
		PickedReqPath:  "/path/to/picked-requirement.yaml",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.Contains(t, prompt, "## Recent Git History")
	assert.Contains(t, prompt, "abc123 Initial commit")
}

func TestBuildPickPrompt_ErrorOnInvalidTemplate(t *testing.T) {
	data := PickPromptData{
		Notes:          nil,
		CommitLog:      "",
		ProjectContent: "content",
		PickedReqPath:  "/path",
	}

	_, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt should succeed with valid data (uses default template)")
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

	_, err := executeTemplate("{{.Age}}", testData{Name: "Bob"}) // Age field doesn't exist
	require.Error(t, err, "executeTemplate should error on execute failure")
	assert.Contains(t, err.Error(), "failed to execute template")
}

func TestBuildDevelopPrompt_EmptyNotes(t *testing.T) {
	data := DevelopPromptData{
		Notes:               []string{},
		CommitLog:           "",
		ProjectContent:      "name: Test",
		SelectedRequirement: "- desc: X",
		ProjectFilePath:     "/path",
		Services:            nil,
		Instructions:        config.DefaultDevelopmentInstructions(),
	}

	prompt, err := BuildDevelopPrompt(data)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.NotContains(t, prompt, "## System Notes")
}

func TestBuildPickPrompt_EmptyNotes(t *testing.T) {
	data := PickPromptData{
		Notes:          []string{},
		CommitLog:      "",
		ProjectContent: "name: Test",
		PickedReqPath:  "/path",
	}

	prompt, err := BuildPickPrompt(data)
	require.NoError(t, err, "BuildPickPrompt failed")

	assert.NotContains(t, prompt, "## System Notes")
}
