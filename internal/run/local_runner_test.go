package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
)

func TestNewLocalRunnerIsNotNil(t *testing.T) {
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	require.NotNil(t, runner)
}
