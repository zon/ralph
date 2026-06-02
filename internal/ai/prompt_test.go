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
	tests := []struct {
		name    string
		svc     config.Service
		svcErr  error
		wantErr bool
		check   func(t *testing.T, prompt string)
	}{
		{
			name: "happy path",
			svc: config.Service{
				Name:    "test-service",
				Command: "myapp",
				Args:    []string{"--port", "8080"},
				Port:    8080,
			},
			svcErr: fmt.Errorf("failed to start service test-service: connection refused"),
			check: func(t *testing.T, prompt string) {
				assert.True(t, strings.Contains(prompt, "Service Startup Failed"))
				assert.True(t, strings.Contains(prompt, "failed to start service test-service"))
				assert.True(t, strings.Contains(prompt, "test-service"))
				assert.True(t, strings.Contains(prompt, "myapp --port 8080"))
				assert.True(t, strings.Contains(prompt, "port 8080"))
				assert.False(t, strings.Contains(prompt, "## Project Requirements"))
				assert.False(t, strings.Contains(prompt, "**Recent Git History:**"))
				assert.True(t, strings.Contains(prompt, "report.md"))
			},
		},
		{
			name: "no port",
			svc: config.Service{
				Name:    "worker",
				Command: "worker",
				Args:    []string{"--config", "worker.yaml"},
			},
			svcErr: fmt.Errorf("failed to start service worker: exit status 1"),
			check: func(t *testing.T, prompt string) {
				assert.True(t, strings.Contains(prompt, "worker --config worker.yaml"))
				assert.False(t, strings.Contains(prompt, "Health check"))
			},
		},
		{
			name:    "should not error with service error",
			svc:     config.Service{Name: "test", Command: "test"},
			svcErr:  fmt.Errorf("service failed"),
			wantErr: false,
		},
		{
			name:    "should not error with plain error",
			svc:     config.Service{Name: "test", Command: "test"},
			svcErr:  errors.New("test error"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testutil.NewContext()
			prompt, err := BuildFixServicePrompt(ctx, tt.svc, tt.svcErr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err, "BuildFixServicePrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildDevelopPrompt(t *testing.T) {
	tests := []struct {
		name    string
		data    DevelopPromptData
		wantErr bool
		errMsg  string
		check   func(t *testing.T, prompt string)
	}{
		{
			name: "base case",
			data: DevelopPromptData{
				Notes:               nil,
				CommitLog:           "",
				ProjectContent:      "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false",
				SelectedRequirement: "- slug: feature-x\n  description: Feature X\n  passing: false",
				ProjectFilePath:     "/path/to/project.yaml",
				Services:            nil,
				Instructions:        config.DefaultDevelopmentInstructions(),
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "# Development Agent")
				assert.Contains(t, prompt, "## Context")
				assert.Contains(t, prompt, "**Selected Requirement:**")
				assert.Contains(t, prompt, "## Instructions")
				assert.Contains(t, prompt, "Feature X")
				assert.Contains(t, prompt, "/path/to/project.yaml")
				assert.Contains(t, prompt, "Implement the selected requirement")
				assert.NotContains(t, prompt, "**Recent Git History:**")
				assert.NotContains(t, prompt, "**System Notes:**")
			},
		},
		{
			name: "with notes",
			data: DevelopPromptData{
				Notes:               []string{"Note 1", "Note 2"},
				CommitLog:           "",
				ProjectContent:      "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false",
				SelectedRequirement: "- slug: feature-x\n  description: Feature X\n  passing: false",
				ProjectFilePath:     "/path/to/project.yaml",
				Services:            nil,
				Instructions:        config.DefaultDevelopmentInstructions(),
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "**System Notes:**")
				assert.Contains(t, prompt, "Note 1")
				assert.Contains(t, prompt, "Note 2")
			},
		},
		{
			name: "with commit log",
			data: DevelopPromptData{
				Notes:               nil,
				CommitLog:           "abc123 Feature A\ndef456 Feature B",
				ProjectContent:      "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false",
				SelectedRequirement: "- slug: feature-x\n  description: Feature X\n  passing: false",
				ProjectFilePath:     "/path/to/project.yaml",
				Services:            nil,
				Instructions:        config.DefaultDevelopmentInstructions(),
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "**Recent Git History:**")
				assert.Contains(t, prompt, "abc123 Feature A")
				assert.Contains(t, prompt, "def456 Feature B")
			},
		},
		{
			name: "with services",
			data: DevelopPromptData{
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
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "**Services**")
				assert.Contains(t, prompt, "api.log")
				assert.Contains(t, prompt, "worker.log")
			},
		},
		{
			name: "with custom instructions",
			data: DevelopPromptData{
				Notes:               nil,
				CommitLog:           "",
				ProjectContent:      "slug: test-project\ntitle: Test Project",
				SelectedRequirement: "- slug: feature-x\n  description: Feature X",
				ProjectFilePath:     "/path/to/project.yaml",
				Services:            nil,
				Instructions:        "Custom instructions: Do something special",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "Custom instructions: Do something special")
			},
		},
		{
			name: "error on invalid template",
			data: DevelopPromptData{
				Notes:               nil,
				CommitLog:           "",
				ProjectContent:      "content",
				SelectedRequirement: "requirement",
				ProjectFilePath:     "/path",
				Services:            nil,
				Instructions:        "{{.InvalidField}}",
			},
			wantErr: true,
			errMsg:  "failed to execute template",
		},
		{
			name: "empty notes",
			data: DevelopPromptData{
				Notes:               []string{},
				CommitLog:           "",
				ProjectContent:      "slug: test",
				SelectedRequirement: "- slug: x\n  description: X",
				ProjectFilePath:     "/path",
				Services:            nil,
				Instructions:        config.DefaultDevelopmentInstructions(),
			},
			check: func(t *testing.T, prompt string) {
				assert.NotContains(t, prompt, "**System Notes:**")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildDevelopPrompt(tt.data)
			if tt.wantErr {
				require.Error(t, err, "BuildDevelopPrompt should error")
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err, "BuildDevelopPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildPickPrompt(t *testing.T) {
	tests := []struct {
		name  string
		data  PickPromptData
		check func(t *testing.T, prompt string)
	}{
		{
			name: "base case",
			data: PickPromptData{
				Notes:          nil,
				CommitLog:      "",
				ProjectContent: "slug: test-project\ntitle: Test Project\nrequirements:\n  - slug: feature-x\n    description: Feature X\n    passing: false\n  - slug: bug-y\n    description: Bug Y\n    passing: false",
				PickedReqPath:  "/path/to/picked-requirement.yaml",
			},
			check: func(t *testing.T, prompt string) {
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
			},
		},
		{
			name: "with notes",
			data: PickPromptData{
				Notes:          []string{"Pick this one"},
				CommitLog:      "",
				ProjectContent: "slug: test-project\ntitle: Test Project",
				PickedReqPath:  "/path/to/picked-requirement.yaml",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "**System Notes:**")
				assert.Contains(t, prompt, "Pick this one")
			},
		},
		{
			name: "with commit log",
			data: PickPromptData{
				Notes:          nil,
				CommitLog:      "abc123 Initial commit\ndef456 Add feature",
				ProjectContent: "slug: test-project\ntitle: Test Project",
				PickedReqPath:  "/path/to/picked-requirement.yaml",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "**Recent Git History:**")
				assert.Contains(t, prompt, "abc123 Initial commit")
			},
		},
		{
			name: "empty notes",
			data: PickPromptData{
				Notes:          []string{},
				CommitLog:      "",
				ProjectContent: "slug: test",
				PickedReqPath:  "/path",
			},
			check: func(t *testing.T, prompt string) {
				assert.NotContains(t, prompt, "**System Notes:**")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildPickPrompt(tt.data)
			require.NoError(t, err, "BuildPickPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildPRSummaryPrompt(t *testing.T) {
	tests := []struct {
		name        string
		projectDesc string
		status      string
		baseBranch  string
		commitLog   string
		outputPath  string
		check       func(t *testing.T, prompt string)
	}{
		{
			name:        "happy path",
			projectDesc: "Test Project",
			status:      "✅ Complete",
			baseBranch:  "main",
			commitLog:   "abc123: Initial commit\ndef456: Add feature\n",
			outputPath:  "/tmp/pr-summary.txt",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "PR summary prompt should not be empty")
				assert.Contains(t, prompt, "Test Project")
				assert.Contains(t, prompt, "✅ Complete")
				assert.Contains(t, prompt, "main..HEAD")
				assert.Contains(t, prompt, "abc123: Initial commit")
				assert.Contains(t, prompt, "/tmp/pr-summary.txt")
			},
		},
		{
			name:        "absolute path",
			projectDesc: "My Project",
			status:      "status",
			baseBranch:  "develop",
			commitLog:   "commit log",
			outputPath:  "relative/path.txt",
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("relative/path.txt")
				assert.Contains(t, prompt, absPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildPRSummaryPrompt(tt.projectDesc, tt.status, tt.baseBranch, tt.commitLog, tt.outputPath)
			require.NoError(t, err, "BuildPRSummaryPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildChangelogPrompt(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		check      func(t *testing.T, prompt string)
	}{
		{
			name:       "happy path",
			outputPath: "/tmp/report.md",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "changelog prompt should not be empty")
				assert.Contains(t, prompt, "report.md")
				assert.Contains(t, prompt, "git diff")
			},
		},
		{
			name:       "absolute path",
			outputPath: "changelog.txt",
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("changelog.txt")
				assert.Contains(t, prompt, absPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildChangelogPrompt(tt.outputPath)
			require.NoError(t, err, "BuildChangelogPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildReviewPRBodyPrompt(t *testing.T) {
	tests := []struct {
		name         string
		reviewName   string
		description  string
		requirements []string
		outputPath   string
		check        func(t *testing.T, prompt string)
	}{
		{
			name:         "happy path",
			reviewName:   "review-2026-03-22",
			description:  "Code review for authentication",
			requirements: []string{"- **security**: JWT validation (✅ Passing)", "- **style**: naming conventions (❌ Not passing)"},
			outputPath:   "/tmp/pr-body.txt",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "PR body prompt should not be empty")
				assert.Contains(t, prompt, "review-2026-03-22")
				assert.Contains(t, prompt, "Code review for authentication")
				assert.Contains(t, prompt, "JWT validation")
				assert.Contains(t, prompt, "/tmp/pr-body.txt")
			},
		},
		{
			name:         "no description",
			reviewName:   "review-2026-03-22",
			description:  "",
			requirements: []string{"- **security**: JWT validation (✅ Passing)"},
			outputPath:   "/tmp/pr-body.txt",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "PR body prompt should not be empty")
				assert.Contains(t, prompt, "review-2026-03-22")
				assert.NotContains(t, prompt, "Description:")
			},
		},
		{
			name:         "absolute path",
			reviewName:   "review",
			description:  "description",
			requirements: []string{"req1", "req2"},
			outputPath:   "relative/path.txt",
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("relative/path.txt")
				assert.Contains(t, prompt, absPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildReviewPRBodyPrompt(tt.reviewName, tt.description, tt.requirements, tt.outputPath)
			require.NoError(t, err, "BuildReviewPRBodyPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildArchitecturePrompt(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		check      func(t *testing.T, prompt string)
	}{
		{
			name:       "happy path",
			outputPath: "/tmp/architecture.yaml",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "architecture prompt should not be empty")
				assert.Contains(t, prompt, "architecture.yaml")
				assert.Contains(t, prompt, "software architect")
				assert.Contains(t, prompt, "domain function")
				assert.Contains(t, prompt, "Major Feature")
				assert.Contains(t, prompt, "cmd/")
				assert.Contains(t, prompt, "internal/")
				assert.Contains(t, prompt, "/tmp/architecture.yaml")
			},
		},
		{
			name:       "absolute path",
			outputPath: "architecture.yaml",
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("architecture.yaml")
				assert.Contains(t, prompt, absPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildArchitecturePrompt(tt.outputPath)
			require.NoError(t, err, "BuildArchitecturePrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildArchitectureFixPrompt(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		errors     []string
		check      func(t *testing.T, prompt string)
	}{
		{
			name:       "happy path",
			outputPath: "/tmp/architecture.yaml",
			errors:     []string{"app 'ralph' is missing description", "module 'internal/ai' must have type domain or implementation"},
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "architecture fix prompt should not be empty")
				assert.Contains(t, prompt, "/tmp/architecture.yaml")
				assert.Contains(t, prompt, "validation errors")
				assert.Contains(t, prompt, "app 'ralph' is missing description")
				assert.Contains(t, prompt, "module 'internal/ai' must have type domain or implementation")
				assert.Contains(t, prompt, "## Errors")
			},
		},
		{
			name:       "absolute path",
			outputPath: "architecture.yaml",
			errors:     []string{"test error"},
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("architecture.yaml")
				assert.Contains(t, prompt, absPath)
			},
		},
		{
			name:       "empty errors",
			outputPath: "/tmp/architecture.yaml",
			errors:     []string{},
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "architecture fix prompt should not be empty")
				assert.Contains(t, prompt, "/tmp/architecture.yaml")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildArchitectureFixPrompt(tt.outputPath, tt.errors)
			require.NoError(t, err, "BuildArchitectureFixPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildProjectFixPrompt(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		projectErr error
		check      func(t *testing.T, prompt string)
	}{
		{
			name:       "happy path",
			outputPath: "/tmp/project.yaml",
			projectErr: errors.New("yaml: line 42: did not find expected node"),
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "project fix prompt should not be empty")
				assert.Contains(t, prompt, "/tmp/project.yaml")
				assert.Contains(t, prompt, "yaml: line 42: did not find expected node")
				assert.Contains(t, prompt, "## Load Error")
			},
		},
		{
			name:       "absolute path",
			outputPath: "project.yaml",
			projectErr: errors.New("test load error"),
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("project.yaml")
				assert.Contains(t, prompt, absPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildProjectFixPrompt(tt.outputPath, tt.projectErr)
			require.NoError(t, err, "BuildProjectFixPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestExecuteTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		data     interface{}
		wantErr  bool
		errMsg   string
		expected string
	}{
		{
			name:     "success",
			tmpl:     "Name: {{.Name}}, Age: {{.Age}}",
			data:     struct{ Name string; Age int }{Name: "Alice", Age: 30},
			expected: "Name: Alice, Age: 30",
		},
		{
			name:    "parse error",
			tmpl:    "{{.Invalid",
			data:    nil,
			wantErr: true,
			errMsg:  "failed to parse template",
		},
		{
			name: "execute error",
			tmpl: "{{.Age}}",
			data: struct{ Name string }{Name: "Bob"},
			wantErr: true,
			errMsg:  "failed to execute template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeTemplate(tt.tmpl, tt.data)
			if tt.wantErr {
				require.Error(t, err, "executeTemplate should error")
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err, "executeTemplate failed")
			assert.Equal(t, tt.expected, result)
		})
	}
}
