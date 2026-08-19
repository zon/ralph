package validate

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
)

// TestAgentClientFixProjectRunsLocally covers the item that the fix loop's
// agent runs locally on the current machine through the same opencode client a
// local run uses, never delegated to a remote runner: FixProject invokes the
// opencode client's RunAgent with a prompt carrying the file path and the parse
// error.
func TestAgentClientFixProjectRunsLocally(t *testing.T) {
	ctx := execcontext.NewContext()

	var capturedModel, capturedPrompt string
	mockOC := &opencode.MockOC{
		RunAgentFunc: func(_ context.Context, model, variant, agent, prompt string) error {
			capturedModel = model
			capturedPrompt = prompt
			return nil
		},
	}

	client := &agentClient{ctx: ctx, oc: mockOC}
	err := client.FixProject("/workspace/repo/projects/project.yaml", &mockParseError{msg: "boom"}, "validate-model")
	require.NoError(t, err)

	require.Equal(t, "validate-model", capturedModel)
	require.Contains(t, capturedPrompt, "/workspace/repo/projects/project.yaml")
	require.Contains(t, capturedPrompt, "boom")
	require.False(t, strings.Contains(capturedPrompt, "argo"), "the agent must not be delegated to a remote runner")
}
