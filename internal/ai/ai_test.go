package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLoopItemPrompt(t *testing.T) {
	t.Run("renders template with FunctionName and FunctionPath", func(t *testing.T) {
		content := "Review {{.FunctionName}} in {{.FunctionPath}}"
		result, err := BuildLoopItemPrompt(content, "DoThing", "internal/pkg/pkg.go")
		require.NoError(t, err)
		assert.Equal(t, "Review DoThing in internal/pkg/pkg.go", result)
	})

	t.Run("malformed template returns error", func(t *testing.T) {
		content := "Review {{.FunctionName} in {{.FunctionPath}}"
		_, err := BuildLoopItemPrompt(content, "DoThing", "internal/pkg/pkg.go")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse template")
	})
}
