package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestAgentClientIsFatal(t *testing.T) {
	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, &opencode.MockOC{})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, client.IsFatal(nil))
	})

	t.Run("detects Insufficient Balance", func(t *testing.T) {
		err := errors.New("opencode execution failed: Insufficient Balance")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects lowercase insufficient balance", func(t *testing.T) {
		err := errors.New("opencode execution failed: insufficient balance")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects billing error", func(t *testing.T) {
		err := errors.New("opencode execution failed: billing error")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects account error", func(t *testing.T) {
		err := errors.New("opencode execution failed: account error")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects payment required", func(t *testing.T) {
		err := errors.New("opencode execution failed: payment required")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects quota exceeded", func(t *testing.T) {
		err := errors.New("opencode execution failed: quota exceeded")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, client.IsFatal(err))
	})
}

func TestAgentClientPickAndDevelop_MockAI(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	projectYAML := `slug: test-project
title: Test project
requirements:
  - slug: req-1
    description: Test requirement
    items:
      - Item 1
    passing: false
`
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := execcontext.NewContext()
	ctx.SetProjectFile("test-project.yaml")

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			// Simulate mock agent writing picked-requirement.yaml for pick prompts
			if strings.Contains(strings.ToLower(prompt), "picked-requirement") {
				pickedReqPath := filepath.Join(filepath.Dir(ctx.ProjectFile()), "picked-requirement.yaml")
				mockReqContent := `- slug: mock-requirement
  description: Mock requirement
  items:
    - Mock item
  passing: false
`
				if err := os.WriteFile(pickedReqPath, []byte(mockReqContent), 0644); err != nil {
					return fmt.Errorf("mock AI failed to write picked-requirement.yaml: %w", err)
				}
			}
			// Append mock modification to project file for develop prompts
			absProjectFile := ctx.ProjectFile()
			if absProjectFile != "" {
				f, err := os.OpenFile(absProjectFile, os.O_APPEND|os.O_WRONLY, 0644)
				if err == nil {
					defer f.Close()
					if _, err := f.WriteString("\n# mock modification"); err != nil {
						return fmt.Errorf("mock AI failed to append to project file: %w", err)
					}
				}
			}
			return nil
		},
	}
	client := NewAgentClient(ctx, mockOC)

	proj := &project.Project{Slug: "test-project", MaxIterations: 1}
	req, err := client.RunPicker(proj)
	require.NoError(t, err)
	require.NotEmpty(t, req)

	err = client.RunDeveloper(proj, req)
	require.NoError(t, err)
}

func TestAgentClientImplementsInterface(t *testing.T) {
	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, &opencode.MockOC{})
	require.NotNil(t, client)
	var _ orchestrationRun.AIClient = client
}

func TestAgentClientPrintStatsDoesNotPanicOnError(t *testing.T) {
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	client := NewAgentClient(ctx, &opencode.MockOC{})
	require.NotPanics(t, func() { client.PrintStats() })
}

func TestAgentClientPrintStatsUsesStoredOCClient(t *testing.T) {
	var called bool
	mockOC := &opencode.MockOC{
		GetStatsFunc: func() (opencode.Stats, error) {
			called = true
			return opencode.Stats{}, nil
		},
	}
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	client := NewAgentClient(ctx, mockOC)
	client.PrintStats()
	assert.True(t, called, "PrintStats should call GetStats on the stored OCClient")
}
