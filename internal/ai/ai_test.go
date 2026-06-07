package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
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
