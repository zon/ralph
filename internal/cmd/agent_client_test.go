package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/config"
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{Slug: "test-project", Path: "projects/test.yaml", Items: project.NewItems([]any{map[string]any{"slug": "csv-serializer", "description": "CSV serializer"}})}
	err := client.RunDeveloper(proj, proj.Items[0])
	require.NoError(t, err)

	assert.Contains(t, developPrompt, "one item of this project")
	assert.Contains(t, developPrompt, "test-project-"+proj.Items[0].Hash())
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
	assert.Contains(t, developPrompt, "`csv-export-"+proj.Items[2].Hash()+"`", "the prompt instructs the exact trailer line")
}

func TestAgentClientDevelopPromptKeylessItemUsesBareTrailer(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	var developPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			developPrompt = prompt
			return nil
		},
	}
	client := NewAgentClient(execcontext.NewContext(), mockOC)

	proj := &project.Project{Slug: "csv-export", Path: "projects/csv-export.yaml", Items: project.NewItems([]any{"exporter", "importer", "writer"})}
	err := client.RunDeveloper(proj, proj.Items[2])
	require.NoError(t, err)

	assert.Contains(t, developPrompt, "(index 2)", "a plain string item is supplied with its index only")
	assert.Contains(t, developPrompt, "`csv-export-"+proj.Items[2].Hash()+"`", "a keyless item still uses the bare branch-hash trailer")
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
				RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
					developPrompt = prompt
					return nil
				},
			}
			client := NewAgentClient(execcontext.NewContext(), mockOC)

			proj := &project.Project{Slug: "csv-export", Path: "projects/csv-export.yaml", Items: project.NewItems([]any{"exporter", "importer", tc.value})}
			item := proj.Items[2]
			err := client.RunDeveloper(proj, item)
			require.NoError(t, err)

			assert.Contains(t, developPrompt, "`"+trailer.Format(proj.Slug, item.Hash())+"`", "the trailer is produced by the shared trailer formatter")
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
`

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			assert.Contains(t, prompt, "orchestration file")
			assert.Contains(t, prompt, "orchestration.md")
			assert.Contains(t, prompt, "project format document installed in the repository")
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
`

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			assert.Contains(t, prompt, "specification file")
			assert.Contains(t, prompt, "spec.md")
			assert.Contains(t, prompt, "orchestration.md")
			assert.Contains(t, prompt, "project format document installed in the repository")
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
`
	newYAML := `slug: new-project
title: New Project
requirements:
  - slug: req-1
    description: New requirement
    items:
      - Item 1
`

	require.NoError(t, os.WriteFile("projects/old.yaml", []byte(oldYAML), 0644))

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
`

	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
			return os.WriteFile("projects/generated.yaml", []byte(projectYAML), 0644)
		},
	}

	client := NewAgentClient(ctx, mockOC)
	input := project.ForOrchestrationInput("specs/features/test/orchestration.md")
	_, err := client.WriteProject(input)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "project format document installed in the repository")
}

func TestAgentClientWriteOrchestrationWithSpecInput(t *testing.T) {
	var promptUsed string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
	assert.Contains(t, promptUsed, "orchestration format document installed in the repository")
}

func TestAgentClientWriteOrchestrationFailureReturnsError(t *testing.T) {
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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
		RunAgentFunc: func(_ context.Context, _, _, _, prompt string) error {
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

// appendAgentToConfig appends `agent: build` to the .ralph/config.yaml created
// by testutil.CreateRalphConfig so the config fallback resolves an agent.
func appendAgentToConfig(t *testing.T, workDir string) {
	t.Helper()
	configPath := filepath.Join(workDir, ".ralph", "config.yaml")
	configData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, append(configData, []byte("agent: build\n")...), 0644))
}

// newNeverPassesAgentClient sets up a temporary working directory with a ralph
// config and returns an AgentClient wired to an opencode mock that captures the
// agent argument. initGit initializes a git repo for the tests that need one:
// the picker reads the commit log, and changelog generation resolves the repo
// root for its temp file. When runAgent is non-nil, the mock calls it with the
// prompt so tests can write the files their prompts expect. runCommand is the
// equivalent hook for the RunCommand path used by changelog generation.
func newNeverPassesAgentClient(t *testing.T, flagAgent string, appendConfigAgent, initGit bool, runAgent func(prompt string) error, runCommand func(prompt string) error) (*AgentClient, *string) {
	t.Helper()

	workDir := t.TempDir()
	t.Chdir(workDir)

	if initGit {
		testutil.InitGitRepo(t, workDir)
		testutil.MakeInitialCommit(t, workDir)
	}

	testutil.CreateRalphConfig(t, workDir)
	if appendConfigAgent {
		appendAgentToConfig(t, workDir)
	}

	ctx := execcontext.NewContext()
	if flagAgent != "" {
		ctx.SetAgent(flagAgent)
	}

	var capturedAgent string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, agent, prompt string) error {
			capturedAgent = agent
			if runAgent != nil {
				return runAgent(prompt)
			}
			return nil
		},
		RunCommandFunc: func(_ context.Context, _, _, agent, prompt string, _, _ io.Writer) error {
			capturedAgent = agent
			if runCommand != nil {
				return runCommand(prompt)
			}
			return nil
		},
	}

	return NewAgentClient(ctx, mockOC), &capturedAgent
}

// writeChangelogOutput writes content to the file path named in the changelog prompt.
func writeChangelogOutput(prompt, content string) error {
	prefix := "Write the changelog entry to the file: "
	idx := strings.Index(prompt, prefix)
	if idx < 0 {
		return errors.New("changelog output file path not found in prompt")
	}
	rest := prompt[idx+len(prefix):]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	path := strings.TrimSpace(rest)
	return os.WriteFile(path, []byte(content), 0644)
}

func TestAgentClientRunPickerNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name              string
		flagAgent         string
		appendConfigAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer", appendConfigAgent: false},
		{name: "config agent set only", appendConfigAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", appendConfigAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, capturedAgent := newNeverPassesAgentClient(t, tc.flagAgent, tc.appendConfigAgent, true, func(_ string) error {
				return os.WriteFile("picked-item-index.txt", []byte("0"), 0644)
			}, nil)

			proj := &project.Project{Slug: "test-project", Items: project.NewItems([]any{"csv-serializer"})}
			_, err := client.RunPicker(proj, proj.Items)
			require.NoError(t, err)
			assert.Equal(t, "", *capturedAgent, "the picker must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

// TestAgentClientGenerateChangelogNeverPassesAgent covers all four branches of
// agent resolution: the changelog prompt produces a supporting artifact and
// must run with opencode's primary agent, never passing --agent. Changelog
// generation needs a git repo for its temp file, so the test passes initGit
// true.
func TestAgentClientGenerateChangelogNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name              string
		flagAgent         string
		appendConfigAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer", appendConfigAgent: false},
		{name: "config agent set only", appendConfigAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", appendConfigAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, capturedAgent := newNeverPassesAgentClient(t, tc.flagAgent, tc.appendConfigAgent, true, nil, func(prompt string) error {
				return writeChangelogOutput(prompt, "changelog content")
			})

			proj := &project.Project{Slug: "test-project", Items: project.NewItems([]any{"csv-serializer"})}
			err := client.GenerateChangelog(proj)
			require.NoError(t, err)
			assert.Equal(t, "", *capturedAgent, "the changelog prompt must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

func TestAgentClientRunDeveloperReceivesConfiguredAgent(t *testing.T) {
	t.Run("flag agent wins", func(t *testing.T) {
		workDir := t.TempDir()
		t.Chdir(workDir)

		testutil.InitGitRepo(t, workDir)
		testutil.MakeInitialCommit(t, workDir)
		testutil.CreateRalphConfig(t, workDir)

		ctx := execcontext.NewContext()
		ctx.SetAgent("code-reviewer")

		var capturedAgent string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, _, _, agent, prompt string) error {
				capturedAgent = agent
				return nil
			},
		}
		client := NewAgentClient(ctx, mockOC)

		proj := &project.Project{Slug: "test-project", Items: project.NewItems([]any{"csv-serializer"})}
		err := client.RunDeveloper(proj, proj.Items[0])
		require.NoError(t, err)
		assert.Equal(t, "code-reviewer", capturedAgent, "item development is code-writing and receives the flag agent")
	})

	t.Run("config agent used when no flag agent set", func(t *testing.T) {
		workDir := t.TempDir()
		t.Chdir(workDir)

		testutil.InitGitRepo(t, workDir)
		testutil.MakeInitialCommit(t, workDir)
		testutil.CreateRalphConfig(t, workDir)
		appendAgentToConfig(t, workDir)

		ctx := execcontext.NewContext()

		var capturedAgent string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, _, _, agent, prompt string) error {
				capturedAgent = agent
				return nil
			},
		}
		client := NewAgentClient(ctx, mockOC)

		proj := &project.Project{Slug: "test-project", Items: project.NewItems([]any{"csv-serializer"})}
		err := client.RunDeveloper(proj, proj.Items[0])
		require.NoError(t, err)
		assert.Equal(t, "build", capturedAgent, "item development is code-writing and falls back to the config agent")
	})
}

// TestAgentClientWriteOrchestrationNeverPassesAgent covers all four branches of
// agent resolution.
func TestAgentClientWriteOrchestrationNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name              string
		flagAgent         string
		appendConfigAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer", appendConfigAgent: false},
		{name: "config agent set only", appendConfigAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", appendConfigAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, capturedAgent := newNeverPassesAgentClient(t, tc.flagAgent, tc.appendConfigAgent, false, nil, nil)

			input := project.ForSpecInput("specs/features/test/spec.md")
			err := client.WriteOrchestration(input)
			require.NoError(t, err)
			assert.Equal(t, "", *capturedAgent, "orchestration generation must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

// TestAgentClientWriteProjectNeverPassesAgent covers all four branches of agent
// resolution.
func TestAgentClientWriteProjectNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name              string
		flagAgent         string
		appendConfigAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer", appendConfigAgent: false},
		{name: "config agent set only", appendConfigAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", appendConfigAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectYAML := `slug: test-project
title: Test Project
requirements:
  - slug: req-1
    description: Test requirement
    items:
      - Item 1
`

			client, capturedAgent := newNeverPassesAgentClient(t, tc.flagAgent, tc.appendConfigAgent, false, func(_ string) error {
				return os.WriteFile("projects/generated.yaml", []byte(projectYAML), 0644)
			}, nil)
			require.NoError(t, os.MkdirAll("projects", 0755))

			input := project.ForSpecInput("specs/features/test/spec.md")
			path, err := client.WriteProject(input)
			require.NoError(t, err)
			assert.Equal(t, "projects/generated.yaml", path)
			assert.Equal(t, "", *capturedAgent, "project generation must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

func TestAgentClientFixServiceStartupReceivesConfiguredAgent(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)
	appendAgentToConfig(t, workDir)

	ctx := execcontext.NewContext()
	ctx.SetAgent("code-reviewer")

	var capturedAgent string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, _, _, agent, prompt string) error {
			capturedAgent = agent
			return nil
		},
	}
	client := NewAgentClient(ctx, mockOC)

	cfg := &config.RalphConfig{Services: []config.Service{{Name: "missing-svc", Command: "definitely-not-a-real-command-xyz"}}}
	err := client.FixServiceStartup(cfg, errors.New("ignored"))
	require.NoError(t, err)
	assert.Equal(t, "code-reviewer", capturedAgent, "service-startup fixes are code-writing and receive the flag agent")
}
