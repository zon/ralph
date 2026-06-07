package ai

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/output"
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

	t.Run("malformed template returns error", func(t *testing.T) {
		content := "Review {{.FunctionName} in {{.FunctionPath}}"
		_, err := BuildLoopItemPrompt(content, "DoThing", "internal/pkg/pkg.go")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse template")
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

func TestRunAgent(t *testing.T) {
	t.Run("captures resolved model, variant, and prompt", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetModel("gpt-4")
		ctx.SetVariant("custom-variant")

		var capturedModel, capturedVariant, capturedPrompt string
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
				capturedModel = model
				capturedVariant = variant
				capturedPrompt = prompt
				return nil
			},
		}

		err := RunAgent(ctx, mockOC, "test prompt")
		require.NoError(t, err)
		assert.Equal(t, "gpt-4", capturedModel)
		assert.Equal(t, "custom-variant", capturedVariant)
		assert.Equal(t, "test prompt", capturedPrompt)
	})

	t.Run("returns underlying error unchanged", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetModel("gpt-4")

		expectedErr := errors.New("agent execution failed")
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
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
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
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
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
				return nil
			},
		}

		err := RunAgent(ctx, mockOC, "quiet prompt")
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
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
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

	t.Run("returns underlying error unchanged", func(t *testing.T) {
		ctx := &execcontext.Context{}

		expectedErr := errors.New("agent execution failed")
		mockOC := &opencode.MockOC{
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
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
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
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
			RunAgentFunc: func(_ context.Context, model, variant, prompt string) error {
				return nil
			},
		}

		err := RunAgentWithModel(ctx, mockOC, "quiet prompt", "some-model")
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}
