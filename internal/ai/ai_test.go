package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/testutil"
)

func TestBuildLoopItemPrompt(t *testing.T) {
	t.Run("renders template with FunctionName and FunctionPath", func(t *testing.T) {
		content := "Review {{.FunctionName}} in {{.FunctionPath}}"
		result, err := BuildLoopItemPrompt(content, "DoThing", "internal/pkg/pkg.go")
		require.NoError(t, err)
		assert.Contains(t, result, "Review DoThing in internal/pkg/pkg.go")
		assert.Contains(t, result, "You are a software architect reviewing source code.")
		assert.Contains(t, result, "Address any issues found")
	})
}

func TestGenerateChangelog(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	t.Chdir(dir)

	errCommandFailed := errors.New("command failed")

	tests := []struct {
		name      string
		setupMock func(*testing.T) *opencode.MockOC
		wantErr   string
		wantErrIs error
	}{
		{
			name: "success renames to report.md and cleans up temp file",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromChangelogPrompt(prompt, "  changelog content\n")
					},
				}
			},
		},
		{
			name: "runcommand error is propagated and temp file cleaned up",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return errCommandFailed
					},
				}
			},
			wantErr:   "opencode execution failed:",
			wantErrIs: errCommandFailed,
		},
		{
			name: "empty summary returns error and temp file cleaned up",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromChangelogPrompt(prompt, "   \n  \n")
					},
				}
			},
			wantErr: "summary file is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove("report.md")

			ctx := &execcontext.Context{}
			mockOC := tt.setupMock(t)
			err := GenerateChangelog(ctx, mockOC)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs), "wrapped error should be reachable via errors.Is")
				}
				_, statErr := os.Stat("report.md")
				assert.True(t, os.IsNotExist(statErr), "report.md should not be created after error")
			} else {
				require.NoError(t, err)
				content, err := os.ReadFile("report.md")
				require.NoError(t, err, "report.md should exist after success")
				assert.Equal(t, "changelog content", strings.TrimSpace(string(content)))
			}

			assertTempFileCleanedUp(t, dir, "changelog")
		})
	}

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromChangelogPrompt(prompt, "result")
			},
		}

		err := GenerateChangelog(ctx, mockOC)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Write a concise changelog entry")
	})
}

func TestResolveModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		setup func(*testing.T)
		want  string
	}{
		{
			name:  "context model overrides config",
			model: "gpt-4",
			want:  "gpt-4",
		},
		{
			name: "falls back to config model",
			want: "claude-3",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("model: claude-3\n"), 0644))
				t.Chdir(dir)
			},
		},
		{
			name: "falls back to default when config load fails",
			want: "deepseek/deepseek-chat",
			setup: func(t *testing.T) {
				t.Chdir(t.TempDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			ctx := &execcontext.Context{}
			if tt.model != "" {
				ctx.SetModel(tt.model)
			}
			result := resolveModel(ctx)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestResolveVariant(t *testing.T) {
	tests := []struct {
		name    string
		variant string
		setup   func(*testing.T)
		want    string
	}{
		{
			name:    "context variant overrides config",
			variant: "custom-variant",
			want:    "custom-variant",
		},
		{
			name: "falls back to config variant",
			want: "sonnet",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("variant: sonnet\n"), 0644))
				t.Chdir(dir)
			},
		},
		{
			name: "falls back to empty when config load fails",
			want: "",
			setup: func(t *testing.T) {
				t.Chdir(t.TempDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			ctx := &execcontext.Context{}
			if tt.variant != "" {
				ctx.SetVariant(tt.variant)
			}
			result := resolveVariant(ctx)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestResolveAgent(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		setup func(*testing.T)
		want  string
	}{
		{
			name:  "context agent overrides config",
			agent: "code-reviewer",
			want:  "code-reviewer",
		},
		{
			name: "falls back to config agent",
			want: "build",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("agent: build\n"), 0644))
				t.Chdir(dir)
			},
		},
		{
			name: "falls back to empty when config load fails",
			want: "",
			setup: func(t *testing.T) {
				t.Chdir(t.TempDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			ctx := &execcontext.Context{}
			if tt.agent != "" {
				ctx.SetAgent(tt.agent)
			}
			result := resolveAgent(ctx)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRunAgent(t *testing.T) {
	t.Run("captures resolved model, variant, agent, and prompt", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetModel("gpt-4")
		ctx.SetVariant("custom-variant")
		ctx.SetAgent("code-reviewer")

		var capturedModel, capturedVariant, capturedAgent, capturedPrompt string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				capturedModel = model
				capturedVariant = variant
				capturedAgent = agent
				capturedPrompt = prompt
				return nil
			},
		}

		err := RunAgent(ctx, mockOC, "test prompt")
		require.NoError(t, err)
		assert.Equal(t, "gpt-4", capturedModel)
		assert.Equal(t, "custom-variant", capturedVariant)
		assert.Equal(t, "code-reviewer", capturedAgent)
		assert.Equal(t, "test prompt", capturedPrompt)
	})

	t.Run("returns underlying error unchanged", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetModel("gpt-4")

		expectedErr := errors.New("agent execution failed")
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return expectedErr
			},
		}

		err := RunAgent(ctx, mockOC, "test prompt")
		assert.Equal(t, expectedErr, err, "error should be returned unchanged, not wrapped")
	})

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return nil
			},
		}

		err := RunAgent(ctx, mockOC, "verbose prompt")
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "verbose prompt")
	})

	t.Run("does not log when not verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(false)
		ctx.SetOutput(output.NewClient(&buf, &buf, false))

		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return nil
			},
		}

		err := RunAgent(ctx, mockOC, "quiet prompt")
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

func TestRunAgentPrimary(t *testing.T) {
	t.Run("resolves model and variant but never passes agent", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("agent: build\n"), 0644))
		t.Chdir(dir)

		ctx := &execcontext.Context{}
		ctx.SetModel("gpt-4")
		ctx.SetVariant("custom-variant")
		ctx.SetAgent("code-reviewer")

		var capturedModel, capturedVariant, capturedAgent, capturedPrompt string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				capturedModel = model
				capturedVariant = variant
				capturedAgent = agent
				capturedPrompt = prompt
				return nil
			},
		}

		err := RunAgentPrimary(ctx, mockOC, "test prompt")
		require.NoError(t, err)
		assert.Equal(t, "gpt-4", capturedModel)
		assert.Equal(t, "custom-variant", capturedVariant)
		assert.Equal(t, "", capturedAgent, "agent must never be passed, even when flag and config resolve one")
		assert.Equal(t, "test prompt", capturedPrompt)
	})

	t.Run("returns underlying error unchanged", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetModel("gpt-4")

		expectedErr := errors.New("agent execution failed")
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return expectedErr
			},
		}

		err := RunAgentPrimary(ctx, mockOC, "test prompt")
		assert.Equal(t, expectedErr, err, "error should be returned unchanged, not wrapped")
	})

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return nil
			},
		}

		err := RunAgentPrimary(ctx, mockOC, "verbose prompt")
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "verbose prompt")
	})

	t.Run("does not log when not verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(false)
		ctx.SetOutput(output.NewClient(&buf, &buf, false))

		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return nil
			},
		}

		err := RunAgentPrimary(ctx, mockOC, "quiet prompt")
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

func TestRunAgentWithModel(t *testing.T) {
	t.Run("passes model verbatim, resolves variant", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetModel("default-model")
		ctx.SetVariant("my-variant")

		var capturedModel, capturedVariant string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				capturedModel = model
				capturedVariant = variant
				return nil
			},
		}

		err := RunAgentWithModel(ctx, mockOC, "test prompt", "explicit-model")
		require.NoError(t, err)
		assert.Equal(t, "explicit-model", capturedModel, "model should be passed verbatim, not resolved")
		assert.Equal(t, "my-variant", capturedVariant, "variant should still be resolved from context")
	})

	t.Run("never passes the configured agent", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("agent: build\n"), 0644))
		t.Chdir(dir)

		ctx := &execcontext.Context{}
		ctx.SetModel("default-model")
		ctx.SetVariant("my-variant")
		ctx.SetAgent("code-reviewer")

		var capturedModel, capturedVariant, capturedAgent string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				capturedModel = model
				capturedVariant = variant
				capturedAgent = agent
				return nil
			},
		}

		err := RunAgentWithModel(ctx, mockOC, "test prompt", "explicit-model")
		require.NoError(t, err)
		assert.Equal(t, "explicit-model", capturedModel, "model should be passed verbatim, not resolved")
		assert.Equal(t, "my-variant", capturedVariant, "variant should still be resolved from context")
		assert.Equal(t, "", capturedAgent, "agent must never be passed, even when flag and config resolve one")
	})

	t.Run("returns underlying error unchanged", func(t *testing.T) {
		ctx := &execcontext.Context{}

		expectedErr := errors.New("agent execution failed")
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return expectedErr
			},
		}

		err := RunAgentWithModel(ctx, mockOC, "test prompt", "some-model")
		assert.Equal(t, expectedErr, err, "error should be returned unchanged, not wrapped")
	})

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return nil
			},
		}

		err := RunAgentWithModel(ctx, mockOC, "verbose prompt", "some-model")
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "verbose prompt")
	})

	t.Run("does not log when not verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(false)
		ctx.SetOutput(output.NewClient(&buf, &buf, false))

		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
				return nil
			},
		}

		err := RunAgentWithModel(ctx, mockOC, "quiet prompt", "some-model")
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

func TestRunOpenCodeAndReadResult(t *testing.T) {
	originalErr := errors.New("command failed")

	tests := []struct {
		name      string
		ctx       *execcontext.Context
		setupMock func(t *testing.T, outputFile string) *opencode.MockOC
		want      string
		wantErr   string
		wantErrIs error
	}{
		{
			name: "success trims leading and trailing whitespace",
			ctx:  &execcontext.Context{},
			setupMock: func(t *testing.T, outputFile string) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return os.WriteFile(outputFile, []byte("  hello world\n  "), 0644)
					},
				}
			},
			want: "hello world",
		},
		{
			name: "runcommand error is wrapped with opencode execution failed",
			ctx:  &execcontext.Context{},
			setupMock: func(t *testing.T, outputFile string) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return originalErr
					},
				}
			},
			wantErr:   "opencode execution failed:",
			wantErrIs: originalErr,
		},
		{
			name: "missing output file returns read error",
			ctx:  &execcontext.Context{},
			setupMock: func(t *testing.T, outputFile string) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return nil
					},
				}
			},
			wantErr: "failed to read output file:",
		},
		{
			name: "whitespace-only output returns summary is empty",
			ctx:  &execcontext.Context{},
			setupMock: func(t *testing.T, outputFile string) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return os.WriteFile(outputFile, []byte("   \n  \n"), 0644)
					},
				}
			},
			wantErr: "summary file is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile := filepath.Join(t.TempDir(), "output.md")
			mockOC := tt.setupMock(t, outputFile)

			result, err := runOpenCodeAndReadResult(tt.ctx, mockOC, "some-model", "some prompt", outputFile)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs), "wrapped error should be reachable via errors.Is")
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRunOpenCodeAndReadResultNeverPassesAgent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("agent: build\n"), 0644))
	t.Chdir(dir)

	ctx := &execcontext.Context{}
	ctx.SetAgent("code-reviewer")

	outputFile := filepath.Join(t.TempDir(), "output.md")
	var capturedAgent string
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
			capturedAgent = agent
			return os.WriteFile(outputFile, []byte("content"), 0644)
		},
	}

	result, err := runOpenCodeAndReadResult(ctx, mockOC, "some-model", "some prompt", outputFile)
	require.NoError(t, err)
	assert.Equal(t, "content", result)
	assert.Equal(t, "", capturedAgent, "agent must never be passed, even when flag and config resolve one")
}

func TestRunOpenCodeAndReadResultVerboseWiring(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		wantNil bool
	}{
		{
			name:    "passes stdout and stderr when verbose",
			verbose: true,
			wantNil: false,
		},
		{
			name:    "passes nil writers when not verbose",
			verbose: false,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &execcontext.Context{}
			ctx.SetVerbose(tt.verbose)
			outputFile := filepath.Join(t.TempDir(), "output.md")

			var capturedStdout, capturedStderr io.Writer
			mockOC := &opencode.MockOC{
				RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
					capturedStdout = stdoutWriter
					capturedStderr = stderrWriter
					return os.WriteFile(outputFile, []byte("content"), 0644)
				},
			}

			_, err := runOpenCodeAndReadResult(ctx, mockOC, "model", "prompt", outputFile)
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, capturedStdout)
				assert.Nil(t, capturedStderr)
			} else {
				assert.Equal(t, os.Stdout, capturedStdout)
				assert.Equal(t, os.Stderr, capturedStderr)
			}
		})
	}
}

func TestCreateTempFile(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	t.Chdir(dir)

	f, err := createTempFile("example.md")
	require.NoError(t, err)
	require.NotNil(t, f)
	defer f.Close()

	expectedPrefix := filepath.Join(dir, "tmp", "example-")
	assert.True(t, strings.HasPrefix(f.Name(), expectedPrefix),
		"expected path starting with %q, got %q", expectedPrefix, f.Name())
	assert.True(t, strings.HasSuffix(f.Name(), ".md"),
		"expected .md extension, got %q", f.Name())

	_, err = os.Stat(filepath.Join(dir, "tmp"))
	assert.NoError(t, err, "tmp/ directory should exist")

	_, err = os.Stat(f.Name())
	assert.NoError(t, err, "file should exist on disk")

	n, err := f.WriteString("test content")
	assert.NoError(t, err)
	assert.Positive(t, n)
}

// writeOutputFromPromptWithPrefix extracts the output file path that follows
// the given marker in the prompt and writes content to it.
func writeOutputFromPromptWithPrefix(prefix string, prompt, content string) error {
	idx := strings.Index(prompt, prefix)
	if idx < 0 {
		return fmt.Errorf("output file path not found in prompt")
	}
	rest := prompt[idx+len(prefix):]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	path := strings.TrimSpace(rest)
	return os.WriteFile(path, []byte(content), 0644)
}

// writeOutputFromChangelogPrompt extracts the output file path from a changelog prompt
// containing "Write the changelog entry to the file: <path>" and writes content to it.
func writeOutputFromChangelogPrompt(prompt, content string) error {
	return writeOutputFromPromptWithPrefix("Write the changelog entry to the file: ", prompt, content)
}

// writeOutputFromPrompt extracts the output file path from a prompt containing
// "Write your summary to the file: <path>" and writes content to it.
func writeOutputFromPrompt(prompt, content string) error {
	return writeOutputFromPromptWithPrefix("Write your summary to the file: ", prompt, content)
}

// writeOutputFromSlugPrompt extracts the output file path from a loop slug
// prompt containing "Write the slug to the file: <path>" and writes content to it.
func writeOutputFromSlugPrompt(prompt, content string) error {
	return writeOutputFromPromptWithPrefix("Write the slug to the file: ", prompt, content)
}

// assertTempFileCleanedUp verifies no files with the given prefix remain in tmp/.
func assertTempFileCleanedUp(t *testing.T, gitRoot, prefix string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(gitRoot, "tmp"))
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix+"-") {
			assert.Fail(t, "temporary file was not cleaned up", "unexpected file: %s", e.Name())
		}
	}
}

func TestGeneratePRSummary(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	t.Chdir(dir)

	errCommandFailed := errors.New("command failed")

	tests := []struct {
		name        string
		projectDesc string
		baseBranch  string
		commitLog   string
		setupMock   func(*testing.T) *opencode.MockOC
		want        string
		wantErr     string
		wantErrIs   error
	}{
		{
			name:        "success returns trimmed summary and cleans up temp file",
			projectDesc: "Test Project",
			baseBranch:  "main",
			commitLog:   "abc: feat\n",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromPrompt(prompt, "  expected summary\n")
					},
				}
			},
			want: "expected summary",
		},
		{
			name:        "runcommand error is wrapped and temp file cleaned up",
			projectDesc: "Test Project",
			baseBranch:  "main",
			commitLog:   "abc: feat\n",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return errCommandFailed
					},
				}
			},
			wantErr:   "opencode execution failed:",
			wantErrIs: errCommandFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &execcontext.Context{}
			mockOC := tt.setupMock(t)
			result, err := GeneratePRSummary(ctx, mockOC, tt.projectDesc, tt.baseBranch, tt.commitLog)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs), "wrapped error should be reachable via errors.Is")
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			assertTempFileCleanedUp(t, dir, "pr-summary")
		})
	}

	t.Run("prompt contains expected inputs", func(t *testing.T) {
		var capturedPrompt string
		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				capturedPrompt = prompt
				return writeOutputFromPrompt(prompt, "result")
			},
		}
		ctx := &execcontext.Context{}
		_, err := GeneratePRSummary(ctx, mockOC, "My Project", "develop", "abc: init\ndef: add\n")
		require.NoError(t, err)
		assert.Contains(t, capturedPrompt, "My Project")
		assert.Contains(t, capturedPrompt, "develop..HEAD")
		assert.Contains(t, capturedPrompt, "abc: init")
		assert.Contains(t, capturedPrompt, "def: add")
	})

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromPrompt(prompt, "result")
			},
		}
		_, err := GeneratePRSummary(ctx, mockOC, "Test", "main", "abc\n")
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Project:")
		assert.Contains(t, buf.String(), "Test")
	})

	t.Run("does not log when not verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(false)
		ctx.SetOutput(output.NewClient(&buf, &buf, false))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromPrompt(prompt, "result")
			},
		}
		_, err := GeneratePRSummary(ctx, mockOC, "Test", "main", "abc\n")
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

func TestGenerateReviewPRBody(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	t.Chdir(dir)

	errCommandFailed := errors.New("command failed")

	tests := []struct {
		name         string
		projectName  string
		projectDesc  string
		requirements []string
		setupMock    func(*testing.T) *opencode.MockOC
		want         string
		wantErr      string
		wantErrIs    error
	}{
		{
			name:         "success returns trimmed body and cleans up temp file",
			projectName:  "my-project",
			projectDesc:  "Test project description",
			requirements: []string{"- **security**: JWT validation", "- **style**: naming"},
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromPrompt(prompt, "  expected review body\n")
					},
				}
			},
			want: "expected review body",
		},
		{
			name:         "runcommand error is wrapped and temp file cleaned up",
			projectName:  "my-project",
			projectDesc:  "Test project description",
			requirements: []string{"- **security**: JWT validation"},
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return errCommandFailed
					},
				}
			},
			wantErr:   "opencode execution failed:",
			wantErrIs: errCommandFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &execcontext.Context{}
			mockOC := tt.setupMock(t)
			result, err := GenerateReviewPRBody(ctx, mockOC, tt.projectName, tt.projectDesc, tt.requirements)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs), "wrapped error should be reachable via errors.Is")
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			assertTempFileCleanedUp(t, dir, "review-pr-body")
		})
	}

	t.Run("prompt contains expected inputs", func(t *testing.T) {
		var capturedPrompt string
		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				capturedPrompt = prompt
				return writeOutputFromPrompt(prompt, "result")
			},
		}
		ctx := &execcontext.Context{}
		_, err := GenerateReviewPRBody(ctx, mockOC, "MyProject", "Auth review", []string{"- **security**: JWT", "- **style**: naming"})
		require.NoError(t, err)
		assert.Contains(t, capturedPrompt, "MyProject")
		assert.Contains(t, capturedPrompt, "Auth review")
		assert.Contains(t, capturedPrompt, "JWT")
		assert.Contains(t, capturedPrompt, "naming")
	})

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromPrompt(prompt, "result")
			},
		}
		_, err := GenerateReviewPRBody(ctx, mockOC, "P", "D", []string{"req1"})
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Review Name:")
		assert.Contains(t, buf.String(), "P")
	})

	t.Run("does not log when not verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(false)
		ctx.SetOutput(output.NewClient(&buf, &buf, false))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromPrompt(prompt, "result")
			},
		}
		_, err := GenerateReviewPRBody(ctx, mockOC, "P", "D", []string{"req1"})
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

// TestGenerateChangelogNeverPassesAgent covers all four branches of agent
// resolution: the changelog prompt produces a supporting artifact and must run
// with opencode's primary agent, never passing --agent.
func TestGenerateChangelogNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name        string
		flagAgent   string
		configAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer"},
		{name: "config agent set only", configAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", configAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.InitGitRepo(t, dir)
			t.Chdir(dir)

			require.NoError(t, os.MkdirAll(".ralph", 0755))
			configContent := "defaultBranch: main\nmodel: deepseek/deepseek-chat\n"
			if tt.configAgent {
				configContent += "agent: build\n"
			}
			require.NoError(t, os.WriteFile(".ralph/config.yaml", []byte(configContent), 0644))

			ctx := &execcontext.Context{}
			if tt.flagAgent != "" {
				ctx.SetAgent(tt.flagAgent)
			}

			var capturedAgent string
			mockOC := &opencode.MockOC{
				RunCommandFunc: func(_ context.Context, _, _, agent, prompt string, _, _ io.Writer) error {
					capturedAgent = agent
					return writeOutputFromChangelogPrompt(prompt, "changelog content")
				},
			}

			err := GenerateChangelog(ctx, mockOC)
			require.NoError(t, err)
			assert.Equal(t, "", capturedAgent, "the changelog prompt must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

// TestGeneratePRSummaryNeverPassesAgent covers all four branches of agent
// resolution: the PR summary prompt produces a supporting artifact and must run
// with opencode's primary agent, never passing --agent.
func TestGeneratePRSummaryNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name        string
		flagAgent   string
		configAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer"},
		{name: "config agent set only", configAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", configAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.InitGitRepo(t, dir)
			t.Chdir(dir)

			require.NoError(t, os.MkdirAll(".ralph", 0755))
			configContent := "defaultBranch: main\nmodel: deepseek/deepseek-chat\n"
			if tt.configAgent {
				configContent += "agent: build\n"
			}
			require.NoError(t, os.WriteFile(".ralph/config.yaml", []byte(configContent), 0644))

			ctx := &execcontext.Context{}
			if tt.flagAgent != "" {
				ctx.SetAgent(tt.flagAgent)
			}

			var capturedAgent string
			mockOC := &opencode.MockOC{
				RunCommandFunc: func(_ context.Context, _, _, agent, prompt string, _, _ io.Writer) error {
					capturedAgent = agent
					return writeOutputFromPrompt(prompt, "summary")
				},
			}

			result, err := GeneratePRSummary(ctx, mockOC, "Test Project", "main", "abc: feat\n")
			require.NoError(t, err)
			assert.Equal(t, "summary", result)
			assert.Equal(t, "", capturedAgent, "the PR summary prompt must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

// TestGenerateReviewPRBodyNeverPassesAgent covers all four branches of agent
// resolution: the PR review body prompt produces a supporting artifact and must
// run with opencode's primary agent, never passing --agent.
func TestGenerateReviewPRBodyNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name        string
		flagAgent   string
		configAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer"},
		{name: "config agent set only", configAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", configAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.InitGitRepo(t, dir)
			t.Chdir(dir)

			require.NoError(t, os.MkdirAll(".ralph", 0755))
			configContent := "defaultBranch: main\nmodel: deepseek/deepseek-chat\n"
			if tt.configAgent {
				configContent += "agent: build\n"
			}
			require.NoError(t, os.WriteFile(".ralph/config.yaml", []byte(configContent), 0644))

			ctx := &execcontext.Context{}
			if tt.flagAgent != "" {
				ctx.SetAgent(tt.flagAgent)
			}

			var capturedAgent string
			mockOC := &opencode.MockOC{
				RunCommandFunc: func(_ context.Context, _, _, agent, prompt string, _, _ io.Writer) error {
					capturedAgent = agent
					return writeOutputFromPrompt(prompt, "review body")
				},
			}

			result, err := GenerateReviewPRBody(ctx, mockOC, "my-project", "Test project description", []string{"- **security**: JWT validation"})
			require.NoError(t, err)
			assert.Equal(t, "review body", result)
			assert.Equal(t, "", capturedAgent, "the PR review body prompt must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}

func TestProposeLoopSlug(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	t.Chdir(dir)

	errCommandFailed := errors.New("command failed")

	tests := []struct {
		name      string
		setupMock func(*testing.T) *opencode.MockOC
		want      string
		wantErr   string
		wantErrIs error
	}{
		{
			name: "success returns trimmed slug and cleans up temp file",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "  fix-formatting\n")
					},
				}
			},
			want: "fix-formatting",
		},
		{
			name: "letters, digits, and hyphens only is usable",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "fmt-vet")
					},
				}
			},
			want: "fmt-vet",
		},
		{
			name: "digits in the slug are usable",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "format42")
					},
				}
			},
			want: "format42",
		},
		{
			name: "runcommand error is wrapped and temp file cleaned up",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return errCommandFailed
					},
				}
			},
			wantErr:   "opencode execution failed:",
			wantErrIs: errCommandFailed,
		},
		{
			name: "whitespace-only output returns no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "   \n  \n")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "output containing a space returns no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "fix formatting")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "repeated dots return no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "fix..format")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "mixed case returns no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "Fix_Formatting")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "slash returns no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "fix/format")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "non-ascii letters return no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "héllo")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "slug starting with a hyphen returns no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "-fix")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
		{
			name: "slug of only hyphens returns no usable slug error",
			setupMock: func(t *testing.T) *opencode.MockOC {
				return &opencode.MockOC{
					RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
						return writeOutputFromSlugPrompt(prompt, "--")
					},
				}
			},
			wantErr: "no usable slug proposed by the AI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &execcontext.Context{}
			mockOC := tt.setupMock(t)
			result, err := ProposeLoopSlug(ctx, mockOC, []string{"run gofmt"})

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs), "wrapped error should be reachable via errors.Is")
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			assertTempFileCleanedUp(t, dir, "loop-slug")
		})
	}

	t.Run("prompt passed to opencode embeds the steps and the output path", func(t *testing.T) {
		var capturedPrompt string
		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				capturedPrompt = prompt
				return writeOutputFromSlugPrompt(prompt, "fix-formatting")
			},
		}
		ctx := &execcontext.Context{}
		_, err := ProposeLoopSlug(ctx, mockOC, []string{"run gofmt", "run go vet"})
		require.NoError(t, err)
		assert.Contains(t, capturedPrompt, "run gofmt")
		assert.Contains(t, capturedPrompt, "run go vet")
		assert.Contains(t, capturedPrompt, filepath.Join(dir, "tmp", "loop-slug-"))
		assert.Contains(t, capturedPrompt, ".md")
	})

	t.Run("logs prompt when verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(true)
		ctx.SetOutput(output.NewClient(&buf, &buf, true))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromSlugPrompt(prompt, "fix-formatting")
			},
		}

		_, err := ProposeLoopSlug(ctx, mockOC, []string{"run gofmt"})
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "run gofmt")
	})

	t.Run("does not log when not verbose", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &execcontext.Context{}
		ctx.SetVerbose(false)
		ctx.SetOutput(output.NewClient(&buf, &buf, false))

		mockOC := &opencode.MockOC{
			RunCommandFunc: func(_ context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
				return writeOutputFromSlugPrompt(prompt, "fix-formatting")
			},
		}

		_, err := ProposeLoopSlug(ctx, mockOC, []string{"run gofmt"})
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

// TestProposeLoopSlugNeverPassesAgent covers all four branches of agent
// resolution: the loop slug prompt produces a supporting artifact and must run
// with opencode's primary agent, never passing --agent.
func TestProposeLoopSlugNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name        string
		flagAgent   string
		configAgent bool
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer"},
		{name: "config agent set only", configAgent: true},
		{name: "flag and config agents set", flagAgent: "code-reviewer", configAgent: true},
		{name: "neither flag nor config agent set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.InitGitRepo(t, dir)
			t.Chdir(dir)

			require.NoError(t, os.MkdirAll(".ralph", 0755))
			configContent := "defaultBranch: main\nmodel: deepseek/deepseek-chat\n"
			if tt.configAgent {
				configContent += "agent: build\n"
			}
			require.NoError(t, os.WriteFile(".ralph/config.yaml", []byte(configContent), 0644))

			ctx := &execcontext.Context{}
			if tt.flagAgent != "" {
				ctx.SetAgent(tt.flagAgent)
			}

			var capturedAgent string
			mockOC := &opencode.MockOC{
				RunCommandFunc: func(_ context.Context, _, _, agent, prompt string, _, _ io.Writer) error {
					capturedAgent = agent
					return writeOutputFromSlugPrompt(prompt, "fix-formatting")
				},
			}

			result, err := ProposeLoopSlug(ctx, mockOC, []string{"run gofmt"})
			require.NoError(t, err)
			assert.Equal(t, "fix-formatting", result)
			assert.Equal(t, "", capturedAgent, "the loop slug prompt must never pass --agent to opencode, so it always runs with the primary agent")
		})
	}
}
