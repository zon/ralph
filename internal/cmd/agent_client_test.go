package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
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

	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, true))

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			// The picker agent writes the selected item's index to disk.
			if strings.Contains(strings.ToLower(prompt), "picker") {
				return os.WriteFile("picked-item-index.txt", []byte("0"), 0644)
			}
			return nil
		},
	}
	client := NewAgentClient(ctx, mockOC)

	proj := &project.Project{Slug: "test-project", Items: project.NewItems([]any{"csv-serializer"})}
	item, err := client.RunPicker(proj, proj.Items)
	require.NoError(t, err)
	require.Equal(t, 0, item.Index)

	err = client.RunDeveloper(proj, item)
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

func TestAgentClientWriteProjectWithOrchestrationInput(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0755))

	projectYAML := `slug: test-project
title: Test Project
requirements:
  - slug: req-1
    description: Test requirement
    items:
      - Item 1
    passing: false
`

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			assert.Contains(t, prompt, "orchestration file")
			assert.Contains(t, prompt, "orchestration.md")
			assert.Contains(t, prompt, "ralph-write-project")
			return os.WriteFile("projects/generated.yaml", []byte(projectYAML), 0644)
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	path, err := client.WriteProject(input)
	require.NoError(t, err)
	assert.Equal(t, "projects/generated.yaml", path)
}

func TestAgentClientWriteProjectWithSpecInput(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0755))

	projectYAML := `slug: test-project
title: Test Project
requirements:
  - slug: req-1
    description: Test requirement
    items:
      - Item 1
    passing: false
`

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			assert.Contains(t, prompt, "specification file")
			assert.Contains(t, prompt, "spec.md")
			assert.Contains(t, prompt, "orchestration.md")
			assert.Contains(t, prompt, "ralph-write-project")
			return os.WriteFile("projects/generated.yaml", []byte(projectYAML), 0644)
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForSpecInput("specs/features/test/spec.md")
	path, err := client.WriteProject(input)
	require.NoError(t, err)
	assert.Equal(t, "projects/generated.yaml", path)
}

func TestAgentClientWriteProjectAgentFailureReturnsError(t *testing.T) {
	ctx := execcontext.NewContext()
	expectedErr := errors.New("agent failed")

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return expectedErr
		},
	}

	client := NewAgentClient(ctx, mockOC)
	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	path, err := client.WriteProject(input)
	require.Error(t, err)
	assert.Empty(t, path)
}

func TestAgentClientWriteProjectNoProjectFileCreatedReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0755))

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return nil
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	path, err := client.WriteProject(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project file found")
	assert.Empty(t, path)
}

func TestAgentClientWriteProjectFindsNewestProjectFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0755))

	oldYAML := `slug: old-project
title: Old Project
requirements:
  - slug: req-1
    description: Old requirement
    items:
      - Item 1
    passing: false
`
	newYAML := `slug: new-project
title: New Project
requirements:
  - slug: req-1
    description: New requirement
    items:
      - Item 1
    passing: false
`

	require.NoError(t, os.WriteFile("projects/old.yaml", []byte(oldYAML), 0644))

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return os.WriteFile("projects/new.yaml", []byte(newYAML), 0644)
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	path, err := client.WriteProject(input)
	require.NoError(t, err)
	assert.Equal(t, "projects/new.yaml", path)
}

func TestAgentClientWriteProjectReturnsPathForUnresolvableFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0755))

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return os.WriteFile("projects/invalid.yaml", []byte("invalid: yaml: ["), 0644)
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	path, err := client.WriteProject(input)
	require.NoError(t, err)
	// WriteProject only reports the generated file's path; resolving it against
	// the run's item query is the caller's job, so an unresolvable file is not
	// an error here.
	assert.Equal(t, "projects/invalid.yaml", path)
}

func TestAgentClientWriteProjectLogsPromptWhenVerbose(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0755))

	var buf bytes.Buffer
	ctx := execcontext.NewContext()
	ctx.SetVerbose(true)
	ctx.SetOutput(output.NewClient(&buf, &buf, true))

	projectYAML := `slug: test-project
title: Test Project
requirements:
  - slug: req-1
    description: Test requirement
    items:
      - Item 1
    passing: false
`

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return os.WriteFile("projects/generated.yaml", []byte(projectYAML), 0644)
		},
	}

	client := NewAgentClient(ctx, mockOC)
	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	_, err := client.WriteProject(input)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "ralph-write-project")
}

func TestAgentClientWriteOrchestrationWithSpecInput(t *testing.T) {
	var promptUsed string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			promptUsed = prompt
			return nil
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForSpecInput("specs/features/test/spec.md")
	err := client.WriteOrchestration(input)
	require.NoError(t, err)
	assert.Contains(t, promptUsed, "specs/features/test/spec.md")
	assert.Contains(t, promptUsed, "orchestration.md")
	assert.Contains(t, promptUsed, "docs/formats/orchestration.md")
}

func TestAgentClientWriteOrchestrationFailureReturnsError(t *testing.T) {
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return errors.New("agent failed")
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForSpecInput("specs/features/test/spec.md")
	err := client.WriteOrchestration(input)
	require.Error(t, err)
}

func TestAgentClientWriteProjectNoProjectsDirReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			return nil
		},
	}

	ctx := execcontext.NewContext()
	client := NewAgentClient(ctx, mockOC)

	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	path, err := client.WriteProject(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read projects directory")
	assert.Empty(t, path)
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
