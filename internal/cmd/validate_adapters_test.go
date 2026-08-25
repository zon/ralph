package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
)

// writeRalphConfig writes a .ralph/config.yaml into a fresh temp working
// directory and changes the working directory to it.
func writeRalphConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	ralphDir := filepath.Join(dir, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(content), 0644))
	t.Chdir(dir)
}

// TestValidateAgentClientFixProjectNeverPassesAgent covers all four branches of
// agent resolution. In each branch the repair prompt runs with opencode's
// primary agent and keeps its model resolution. It still carries the file path
// and the parse error.
func TestValidateAgentClientFixProjectNeverPassesAgent(t *testing.T) {
	tests := []struct {
		name        string
		flagAgent   string
		configAgent string
	}{
		{name: "flag agent set only", flagAgent: "code-reviewer"},
		{name: "config agent set only", configAgent: "build"},
		{name: "flag and config agents set", flagAgent: "code-reviewer", configAgent: "build"},
		{name: "neither flag nor config agent set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configContent := "model: deepseek/deepseek-chat\n"
			if tt.configAgent != "" {
				configContent += "agent: " + tt.configAgent + "\n"
			}
			writeRalphConfig(t, configContent)

			ctx := execcontext.NewContext()
			if tt.flagAgent != "" {
				ctx.SetAgent(tt.flagAgent)
			}

			var capturedModel, capturedAgent, capturedPrompt string
			mockOC := &opencode.MockOC{
				RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
					capturedModel = model
					capturedAgent = agent
					capturedPrompt = prompt
					return nil
				},
			}

			client := &validateAgentClient{ctx: ctx, oc: mockOC}
			err := client.FixProject(anyPath, &mockParseError{msg: "boom"}, "validate-model")
			require.NoError(t, err)

			assert.Equal(t, "validate-model", capturedModel, "model resolution is unchanged: opencode receives the validate-specific model")
			assert.Equal(t, "", capturedAgent, "the validate project-file repair prompt must never pass --agent to opencode")
			assert.Contains(t, capturedPrompt, anyPath)
			assert.Contains(t, capturedPrompt, "boom")
			assert.NotContains(t, capturedPrompt, "argo", "the repair prompt does not reference a remote runner")
		})
	}
}

// TestResolveValidateModelResolvesValidateSpecificModel covers that the
// validate repair prompt keeps its model resolution: when validate.model is set
// in the config, the validator resolves that model.
func TestResolveValidateModelResolvesValidateSpecificModel(t *testing.T) {
	writeRalphConfig(t, "model: main-model\nvalidate:\n  model: validate-model\n")
	require.Equal(t, "validate-model", resolveValidateModel())
}

// TestResolveValidateModelFallsBackToMainModel covers that the validate repair
// prompt keeps its model resolution: without a validate-specific model, the
// top-level config model is the fallback.
func TestResolveValidateModelFallsBackToMainModel(t *testing.T) {
	writeRalphConfig(t, "model: main-model\n")
	require.Equal(t, "main-model", resolveValidateModel())
}

const anyPath = "/workspace/repo/projects/test-project.yaml"

type mockParseError struct {
	msg string
}

func (e *mockParseError) Error() string {
	return e.msg
}
