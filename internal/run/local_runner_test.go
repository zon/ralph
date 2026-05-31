package run

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
)

func TestNewLocalRunnerIsNotNil(t *testing.T) {
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	require.NotNil(t, runner)
}

func TestNewLocalRunner_WiresSystemEnvClient(t *testing.T) {
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	_, ok := runner.Env().(*SystemEnvClient)
	require.True(t, ok, "expected runner.env to be *SystemEnvClient")
}

func TestNewLocalRunner_EnvNotInWorkflowByDefault(t *testing.T) {
	os.Unsetenv("RALPH_WORKFLOW_EXECUTION")
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	require.False(t, runner.Env().InWorkflow())
}
