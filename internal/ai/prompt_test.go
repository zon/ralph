package ai

import (
	"errors"
	"fmt"
	"os"
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

func TestBuildPRSummaryPrompt(t *testing.T) {
	tests := []struct {
		name        string
		projectDesc string
		baseBranch  string
		commitLog   string
		usage       string
		outputPath  string
		check       func(t *testing.T, prompt string)
	}{
		{
			name:        "happy path",
			projectDesc: "Test Project",
			baseBranch:  "main",
			commitLog:   "abc123: Initial commit\ndef456: Add feature\n",
			outputPath:  "/tmp/pr-summary.txt",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "PR summary prompt should not be empty")
				assert.Contains(t, prompt, "Test Project")
				assert.Contains(t, prompt, "main..HEAD")
				assert.Contains(t, prompt, "abc123: Initial commit")
				assert.Contains(t, prompt, "/tmp/pr-summary.txt")
			},
		},
		{
			name:        "absolute path",
			projectDesc: "My Project",
			baseBranch:  "develop",
			commitLog:   "commit log",
			outputPath:  "relative/path.txt",
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("relative/path.txt")
				assert.Contains(t, prompt, absPath)
			},
		},
		{
			name:        "usage is embedded when provided",
			projectDesc: "Test Project",
			baseBranch:  "main",
			commitLog:   "abc123: Initial commit\n",
			usage:       "Input tokens: 1.5M, Output tokens: 542.0K, Cost: $12.34",
			outputPath:  "/tmp/pr-summary.txt",
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "## AI Usage")
				assert.Contains(t, prompt, "Input tokens: 1.5M, Output tokens: 542.0K, Cost: $12.34")
				assert.Contains(t, prompt, "Usage\" section")
			},
		},
		{
			name:        "usage is omitted when empty",
			projectDesc: "Test Project",
			baseBranch:  "main",
			commitLog:   "abc123: Initial commit\n",
			outputPath:  "/tmp/pr-summary.txt",
			check: func(t *testing.T, prompt string) {
				assert.NotContains(t, prompt, "## AI Usage")
				assert.NotContains(t, prompt, "Usage\" section")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildPRSummaryPrompt(tt.projectDesc, tt.baseBranch, tt.commitLog, tt.usage, tt.outputPath)
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

func TestBuildChangelogPromptDoesNotInstructCompletionTrailer(t *testing.T) {
	prompt, err := BuildChangelogPrompt("/tmp/report.md")
	require.NoError(t, err)
	assert.NotEmpty(t, prompt, "changelog prompt should not be empty")
	assert.NotContains(t, prompt, "Ralph item")
	assert.NotContains(t, prompt, "completed")
	assert.NotContains(t, prompt, "completion trailer")
	assert.NotContains(t, prompt, "trailer")
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

func TestBuildWriteProjectPrompt(t *testing.T) {
	tests := []struct {
		name  string
		data  WriteProjectPromptData
		check func(t *testing.T, prompt string)
	}{
		{
			name: "orchestration input",
			data: WriteProjectPromptData{
				InputPath: "specs/features/my-feature/orchestration.md",
				InputType: "orchestration file",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "orchestration file")
				assert.Contains(t, prompt, "specs/features/my-feature/orchestration.md")
				assert.Contains(t, prompt, "project format document installed in the repository")
				assert.NotContains(t, prompt, "ralph-write-project")
				assert.NotContains(t, prompt, "docs/formats/")
				assert.NotContains(t, prompt, "Also read the orchestration document")
			},
		},
		{
			name: "spec input with orchestration",
			data: WriteProjectPromptData{
				InputPath:         "specs/features/my-feature/spec.md",
				InputType:         "specification file",
				HasOrchestration:  true,
				OrchestrationPath: "specs/features/my-feature/orchestration.md",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "specification file")
				assert.Contains(t, prompt, "specs/features/my-feature/spec.md")
				assert.Contains(t, prompt, "specs/features/my-feature/orchestration.md")
				assert.Contains(t, prompt, "project format document installed in the repository")
				assert.NotContains(t, prompt, "ralph-write-project")
				assert.NotContains(t, prompt, "docs/formats/")
				assert.Contains(t, prompt, "Also read the orchestration document")
			},
		},
		{
			name: "spec input without orchestration",
			data: WriteProjectPromptData{
				InputPath: "specs/features/my-feature/spec.md",
				InputType: "specification file",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "specification file")
				assert.NotContains(t, prompt, "Also read the orchestration document")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildWriteProjectPrompt(tt.data)
			require.NoError(t, err, "BuildWriteProjectPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildWriteOrchestrationPrompt(t *testing.T) {
	tests := []struct {
		name  string
		data  WriteOrchestrationPromptData
		check func(t *testing.T, prompt string)
	}{
		{
			name: "spec input",
			data: WriteOrchestrationPromptData{
				SpecPath: "specs/features/my-feature/spec.md",
			},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "specs/features/my-feature/spec.md")
				assert.Contains(t, prompt, "orchestration.md")
				assert.Contains(t, prompt, "orchestration format document installed in the repository")
				assert.NotContains(t, prompt, "docs/formats/")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildWriteOrchestrationPrompt(tt.data)
			require.NoError(t, err, "BuildWriteOrchestrationPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildItemDevelopPrompt(t *testing.T) {
	keyed := ItemDevelopPromptData{
		Notes:           nil,
		CommitLog:       "abc123 feat: add exporter\n",
		ProjectContent:  "slug: csv-export\ntitle: CSV Export\ntasks:\n  - slug: exporter\n    description: Exporter\n",
		ItemIndex:       2,
		ItemKey:         "export-endpoint",
		ItemValue:       "slug: export-endpoint\ndescription: Build the export endpoint",
		Trailer:         "csv-export-IYAWN02",
		ProjectFilePath: "projects/csv-export.yaml",
		Services:        nil,
	}

	t.Run("presents the selected item verbatim with its index and key", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "**Selected Item (index 2, key export-endpoint):**")
		assert.Contains(t, prompt, "slug: export-endpoint")
		assert.Contains(t, prompt, "description: Build the export endpoint")
	})

	t.Run("presents a keyless item with its index only", func(t *testing.T) {
		data := keyed
		data.ItemKey = ""
		prompt, err := BuildItemDevelopPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "**Selected Item (index 2):**")
		assert.NotContains(t, prompt, "key export-endpoint")
	})

	t.Run("describes conventional item fields as optional", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "every field is optional")
		assert.Contains(t, prompt, "plain string")
		assert.Contains(t, prompt, "any other shape")
	})

	t.Run("tells the agent the last line must be the completion trailer", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "last line of `report.md` MUST be the completion trailer")
		assert.Contains(t, prompt, "csv-export-IYAWN02")
	})

	t.Run("describes the trailer as a bare branch-hash line", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "`<branch>-<hash>`")
		assert.Contains(t, prompt, "`csv-export-IYAWN02`")
	})

	t.Run("notes that a trailer naming a different branch is not evidence of completion", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "A trailer naming a different branch is not evidence of completion")
	})

	t.Run("instructs the agent not to modify the project file", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "Do not modify the project file")
		assert.Contains(t, prompt, "read-only for the whole run")
	})

	t.Run("tells the agent to leave completion fields alone", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "Do not edit any completion field")
	})

	t.Run("still asks for report.md and blocked.md without a trailer", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "Write a concise report to `report.md`")
		assert.Contains(t, prompt, "`blocked.md`")
		assert.Contains(t, prompt, "no completion trailer")
	})

	t.Run("uses the item-based default instructions when none are supplied", func(t *testing.T) {
		data := keyed
		data.Instructions = ""
		prompt, err := BuildItemDevelopPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "read the selected item carefully")
	})

	t.Run("uses the repository's own instructions when supplied", func(t *testing.T) {
		data := keyed
		data.Instructions = "1. **Ship** — do it the house way."
		prompt, err := BuildItemDevelopPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "do it the house way")
		assert.NotContains(t, prompt, "read the selected item carefully")
	})

	t.Run("explains that the completion trailer is the only way an item completes", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "completion trailer")
		assert.Contains(t, prompt, "only way")
	})

	t.Run("defers the item shape to the repository's installed project format", func(t *testing.T) {
		prompt, err := BuildItemDevelopPrompt(keyed)
		require.NoError(t, err)
		assert.Contains(t, prompt, "project format the repository has installed")
	})
}

func TestBuildItemPickPrompt(t *testing.T) {
	data := ItemPickPromptData{
		Notes:          nil,
		CommitLog:      "abc123 feat: add exporter\n",
		ProjectContent: "slug: csv-export\ntitle: CSV Export\ntasks:\n  - slug: exporter\n    description: Exporter\n  - slug: importer\n    description: Importer\n",
		Items:          "item 1 (exporter):\nslug: exporter\ndescription: Exporter\nitem 3 (importer):\nslug: importer\ndescription: Importer",
	}

	t.Run("frames the agent as an item picker", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "Item Picker Agent")
		assert.Contains(t, prompt, "incomplete item")
	})

	t.Run("carries the full project file, the labelled incomplete items, and the commit log", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "slug: csv-export")
		assert.Contains(t, prompt, "title: CSV Export")
		assert.Contains(t, prompt, "item 1 (exporter):")
		assert.Contains(t, prompt, "item 3 (importer):")
		assert.Contains(t, prompt, "abc123 feat: add exporter")
	})

	t.Run("selects by dependencies, logical ordering, and impact, not array order", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "dependencies between items")
		assert.Contains(t, prompt, "logical ordering")
		assert.Contains(t, prompt, "impact on the overall project")
		assert.Contains(t, prompt, "not constrained to array order")
	})

	t.Run("reports the picked index to a file", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "0-based index")
		assert.Contains(t, prompt, "picked-item-index.txt")
	})

	t.Run("treats the incomplete list as authoritative and forbids auditing wider history", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "incomplete items list as authoritative")
		assert.Contains(t, prompt, "Do not audit the wider git history or the working tree for completion evidence")
		assert.Contains(t, prompt, "Select exactly one of the listed items")
	})

	t.Run("describes trailers as bare branch-hash lines and ignores other branches", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "`<branch>-<hash>`")
		assert.Contains(t, prompt, "A trailer naming a different branch is not evidence of completion")
	})

	t.Run("tells the agent to make no code changes", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "Do not make any code changes")
	})

	t.Run("renders the project file, items, and commit log sections", func(t *testing.T) {
		prompt, err := BuildItemPickPrompt(data)
		require.NoError(t, err)
		assert.Contains(t, prompt, "**Project File:**")
		assert.Contains(t, prompt, "**Incomplete Items:**")
		assert.Contains(t, prompt, "**Recent Git History:**")
		assert.NotContains(t, prompt, "**System Notes:**")
	})
}

func TestEmbeddedPromptsCarryNoRalphOwnedPaths(t *testing.T) {
	// GIVEN the embedded instruction templates
	// WHEN they are searched for docs/formats/
	// THEN no match is found
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		require.NoError(t, err)
		assert.NotContains(t, string(data), "docs/formats/", "%s must not name a ralph-shipped document path", e.Name())
	}
}

func TestDefaultItemDevelopmentInstructions(t *testing.T) {
	instructions := DefaultItemDevelopmentInstructions()
	assert.Contains(t, instructions, "selected item")
	assert.NotContains(t, instructions, "completion trailer")
}

func TestDefaultItemDevelopmentInstructionsDeferToTheRepositoryStandards(t *testing.T) {
	// GIVEN the default development steps
	// WHEN an agent looks for where code belongs and how tests are written
	// THEN it is sent to the repository's own agent instructions and standards
	instructions := DefaultItemDevelopmentInstructions()
	assert.Contains(t, instructions, "repository's agent instructions and the standards they link to")
	assert.Contains(t, instructions, "where code belongs")
	assert.Contains(t, instructions, "how tests are written")
}

func TestDefaultItemDevelopmentInstructionsKeepTestsBeforeCode(t *testing.T) {
	// GIVEN the default development steps
	// WHEN their order is read
	// THEN the tests step precedes the code step and the run ends verified
	instructions := DefaultItemDevelopmentInstructions()
	tests := strings.Index(instructions, "**Tests**")
	code := strings.Index(instructions, "**Code**")
	verify := strings.Index(instructions, "**Verify**")
	require.Greater(t, tests, -1, "the steps must include a tests step")
	assert.Less(t, tests, code, "the tests step must precede the code step")
	assert.Less(t, code, verify, "the code step must precede the verify step")
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
			name: "success",
			tmpl: "Name: {{.Name}}, Age: {{.Age}}",
			data: struct {
				Name string
				Age  int
			}{Name: "Alice", Age: 30},
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
			name:    "execute error",
			tmpl:    "{{.Age}}",
			data:    struct{ Name string }{Name: "Bob"},
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

func TestBuildResolveMergeConflictsPrompt(t *testing.T) {
	prompt, err := BuildResolveMergeConflictsPrompt("main", "feature-branch")
	require.NoError(t, err)
	assert.Contains(t, prompt, "main")
	assert.Contains(t, prompt, "feature-branch")
	assert.Contains(t, prompt, "git merge main")
}

func TestBuildLoopPrompt(t *testing.T) {
	tests := []struct {
		name  string
		steps []string
		check func(t *testing.T, prompt string)
	}{
		{
			name:  "embeds a single step",
			steps: []string{"run gofmt"},
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "loop prompt should not be empty")
				assert.Contains(t, prompt, "- run gofmt")
			},
		},
		{
			name:  "embeds multiple steps in the order given",
			steps: []string{"run gofmt", "run go vet", "run tests"},
			check: func(t *testing.T, prompt string) {
				first := strings.Index(prompt, "run gofmt")
				second := strings.Index(prompt, "run go vet")
				third := strings.Index(prompt, "run tests")
				require.Greater(t, first, -1, "first step must appear in the prompt")
				require.Greater(t, second, -1, "second step must appear in the prompt")
				require.Greater(t, third, -1, "third step must appear in the prompt")
				assert.Less(t, first, second, "first step must appear before the second")
				assert.Less(t, second, third, "second step must appear before the third")
			},
		},
		{
			name:  "empty steps render without error",
			steps: nil,
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "loop prompt should not be empty")
				assert.Contains(t, prompt, "Follow these steps in order:")
				assert.NotContains(t, prompt, "- run gofmt")
			},
		},
		{
			name:  "empty steps slice renders without error",
			steps: []string{},
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "loop prompt should not be empty")
				assert.Contains(t, prompt, "Follow these steps in order:")
				assert.NotContains(t, prompt, "- run gofmt")
			},
		},
		{
			name:  "instructs the agent to write a brief and simple summary to report.md",
			steps: []string{"run gofmt"},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "Write a brief and simple summary of what you did in response to `report.md`")
			},
		},
		{
			name:  "instructs the summary to describe only what was done, not the steps",
			steps: []string{"run gofmt"},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "Do not restate the loop steps")
			},
		},
		{
			name:  "instructs writing exactly NOTHING_TO_DO when nothing was necessary",
			steps: []string{"run gofmt"},
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "write exactly `NOTHING_TO_DO` to `report.md`")
				assert.Contains(t, prompt, "nothing was necessary")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildLoopPrompt(tt.steps)
			require.NoError(t, err, "BuildLoopPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}

func TestBuildLoopSlugPrompt(t *testing.T) {
	tests := []struct {
		name       string
		steps      []string
		outputPath string
		check      func(t *testing.T, prompt string)
	}{
		{
			name:       "embeds steps in the order given",
			steps:      []string{"run gofmt", "run go vet", "run tests"},
			outputPath: "/tmp/loop-slug.txt",
			check: func(t *testing.T, prompt string) {
				assert.NotEmpty(t, prompt, "loop slug prompt should not be empty")
				first := strings.Index(prompt, "run gofmt")
				second := strings.Index(prompt, "run go vet")
				third := strings.Index(prompt, "run tests")
				require.Greater(t, first, -1, "first step must appear in the prompt")
				require.Greater(t, second, -1, "second step must appear in the prompt")
				require.Greater(t, third, -1, "third step must appear in the prompt")
				assert.Less(t, first, second, "first step must appear before the second")
				assert.Less(t, second, third, "second step must appear before the third")
			},
		},
		{
			name:       "resolves the output path to an absolute path",
			steps:      []string{"run gofmt"},
			outputPath: "relative/loop-slug.txt",
			check: func(t *testing.T, prompt string) {
				absPath, _ := filepath.Abs("relative/loop-slug.txt")
				assert.Contains(t, prompt, absPath)
			},
		},
		{
			name:       "instructs the AI to write only the slug",
			steps:      []string{"run gofmt"},
			outputPath: "/tmp/loop-slug.txt",
			check: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "Write the slug to the file: /tmp/loop-slug.txt")
				assert.Contains(t, prompt, "lowercase letters")
				assert.Contains(t, prompt, "hyphens")
				assert.NotContains(t, prompt, "docs/formats/")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := BuildLoopSlugPrompt(tt.steps, tt.outputPath)
			require.NoError(t, err, "BuildLoopSlugPrompt failed")
			if tt.check != nil {
				tt.check(t, prompt)
			}
		})
	}
}
