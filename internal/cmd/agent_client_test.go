package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/projectfile"
	"github.com/zon/ralph/internal/testutil"
	"github.com/zon/ralph/internal/trailer"
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

func TestAgentClientRunPickerGivesOnlyIncompleteItemsEachLabelledWithIndexAndKey(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	var pickPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			pickPrompt = prompt
			return os.WriteFile("picked-item-index.txt", []byte("1"), 0644)
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{
		Slug: "test-project",
		Path: "projects/test.yaml",
		Items: project.NewItems([]any{
			map[string]any{"slug": "one", "description": "first"},
			map[string]any{"slug": "exporter", "description": "export endpoint"},
			map[string]any{"slug": "two", "description": "second"},
			map[string]any{"slug": "importer", "description": "import endpoint"},
		}),
		Doc: &projectfile.Document{
			Raw: "slug: test-project\ntitle: Test Project\nitems:\n" +
				"  - slug: one\n    description: first\n" +
				"  - slug: exporter\n    description: export endpoint\n" +
				"  - slug: two\n    description: second\n" +
				"  - slug: importer\n    description: import endpoint\n",
		},
	}

	item, err := client.RunPicker(proj, []project.Item{proj.Items[1], proj.Items[3]})
	require.NoError(t, err)
	require.Equal(t, 1, item.Index)

	assert.Contains(t, pickPrompt, "slug: test-project", "the full project file is carried in the prompt")
	assert.Contains(t, pickPrompt, "item 1 (exporter):", "the remaining item is labelled with its index and key")
	assert.Contains(t, pickPrompt, "slug: exporter")
	assert.Contains(t, pickPrompt, "item 3 (importer):", "the remaining item is labelled with its index and key")
	assert.Contains(t, pickPrompt, "slug: importer")
	assert.NotContains(t, pickPrompt, "item 0 (", "the complete item is not offered to the picker")
	assert.NotContains(t, pickPrompt, "item 2 (", "the complete item is not offered to the picker")
	assert.NotContains(t, pickPrompt, "passing", "the instructions never mention a passing field")
	assert.NotContains(t, pickPrompt, "requirement", "the instructions describe items, not requirements")
	assert.Contains(t, pickPrompt, "not constrained to array order")
	assert.Contains(t, pickPrompt, "Do not make any code changes")
	assert.Contains(t, pickPrompt, "picked-item-index.txt", "the agent reports the index it selected")
}

func TestAgentClientRunDeveloperUsesItemBasedInstructionsByDefault(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	var developPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{Slug: "test-project", Path: "projects/test.yaml", Items: project.NewItems([]any{map[string]any{"slug": "csv-serializer", "description": "CSV serializer"}})}
	err := client.RunDeveloper(proj, proj.Items[0])
	require.NoError(t, err)

	assert.Contains(t, developPrompt, "one item of this project")
	assert.Contains(t, developPrompt, "Ralph item 0 (csv-serializer) completed")
	assert.NotContains(t, developPrompt, "implementing a specific requirement")
	assert.NotContains(t, developPrompt, "{{.SelectedRequirement}}")
}

func TestAgentClientRunDeveloperHonorsCustomInstructions(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	custom := "Custom instructions: focus on performance"
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".ralph", "instructions.md"), []byte(custom), 0644))

	var developPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{Slug: "test-project", Items: project.NewItems([]any{"csv-serializer"})}
	err := client.RunDeveloper(proj, proj.Items[0])
	require.NoError(t, err)

	assert.Contains(t, developPrompt, custom)
	assert.NotContains(t, developPrompt, "read the selected item carefully")
}

func TestAgentClientRunPickerCarriesFullProjectFileAsContext(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	raw := "title: CSV Export\nnotes:\n  owner: platform\ntasks:\n" +
		"  - slug: exporter\n    description: export endpoint\n" +
		"  - slug: importer\n    description: import endpoint\n"
	var pickPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			pickPrompt = prompt
			return os.WriteFile("picked-item-index.txt", []byte("0"), 0644)
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{
		Slug: "csv-export",
		Path: "projects/csv-export.yaml",
		Items: project.NewItems([]any{
			map[string]any{"slug": "exporter", "description": "export endpoint"},
			map[string]any{"slug": "importer", "description": "import endpoint"},
		}),
		Doc: &projectfile.Document{Raw: raw},
	}

	_, err := client.RunPicker(proj, proj.Items)
	require.NoError(t, err)
	assert.Contains(t, pickPrompt, strings.TrimRight(raw, "\n"), "the whole project file is included in the prompt as context")
	assert.Contains(t, pickPrompt, "owner: platform", "content outside the item array is retained")
	assert.Contains(t, pickPrompt, "title: CSV Export", "content outside the item array is retained")
}

func TestAgentClientDevelopPromptCarriesSelectedItemVerbatim(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	var developPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	values := []any{
		"exporter",
		map[string]any{"slug": "importer", "description": "import endpoint"},
		map[string]any{"slug": "export-endpoint", "description": "Build the export endpoint"},
	}
	proj := &project.Project{Slug: "csv-export", Path: "projects/csv-export.yaml", Items: project.NewItems(values)}
	item := proj.Items[2]
	err := client.RunDeveloper(proj, item)
	require.NoError(t, err)

	rendered, err := yaml.Marshal(item.Value)
	require.NoError(t, err)
	assert.Contains(t, developPrompt, strings.TrimRight(string(rendered), "\n"), "the item's value is included in the prompt exactly as it appears in the resolved array")
}

func TestAgentClientDevelopPromptSuppliesIndexKeyAndTrailer(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	var developPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{
		Slug:  "csv-export",
		Path:  "projects/csv-export.yaml",
		Items: project.NewItems([]any{"exporter", "importer", map[string]any{"slug": "export-endpoint", "description": "build the export endpoint"}}),
	}
	err := client.RunDeveloper(proj, proj.Items[2])
	require.NoError(t, err)

	assert.Contains(t, developPrompt, "(index 2, key export-endpoint)", "the prompt supplies index 2 and the key export-endpoint")
	assert.Contains(t, developPrompt, "`Ralph item 2 (export-endpoint) completed`", "the prompt instructs the exact trailer line")
}

func TestAgentClientDevelopPromptKeylessItemUsesIndexOnlyTrailer(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	var developPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{Slug: "csv-export", Path: "projects/csv-export.yaml", Items: project.NewItems([]any{"exporter", "importer", "writer"})}
	err := client.RunDeveloper(proj, proj.Items[2])
	require.NoError(t, err)

	assert.Contains(t, developPrompt, "(index 2)", "a plain string item is supplied with its index only")
	assert.Contains(t, developPrompt, "`Ralph item 2 completed`", "a keyless item uses the index-only trailer form")
}

func TestAgentClientDevelopPromptTrailerComesFromSharedFormatter(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	for _, tc := range []struct {
		name  string
		value any
	}{
		{name: "keyed", value: map[string]any{"slug": "export-endpoint", "description": "build the export endpoint"}},
		{name: "keyless", value: "plain string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var developPrompt string
			mockOC := &opencode.MockOC{
				RunAgentFunc: func(_ context.Context, _, _, prompt string) error {
					developPrompt = prompt
					return nil
				},
			}
			client := NewAgentClient(execcontext.NewContext(), mockOC)

			proj := &project.Project{Slug: "csv-export", Path: "projects/csv-export.yaml", Items: project.NewItems([]any{"exporter", "importer", tc.value})}
			item := proj.Items[2]
			err := client.RunDeveloper(proj, item)
			require.NoError(t, err)

			assert.Contains(t, developPrompt, "`"+trailer.Format(item.Index, item.Key())+"`", "the trailer is produced by the shared trailer formatter")
		})
	}
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
