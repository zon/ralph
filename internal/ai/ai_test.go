package ai

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFatalError(t *testing.T) {
	t.Run("nil error returns false", func(t *testing.T) {
		assert.False(t, IsFatalError(nil))
	})

	t.Run("generic billing patterns are fatal", func(t *testing.T) {
		assert.True(t, IsFatalError(fmt.Errorf("billing failure")))
		assert.True(t, IsFatalError(fmt.Errorf("account suspended")))
		assert.True(t, IsFatalError(fmt.Errorf("payment required")))
	})

	t.Run("anthropic patterns are fatal", func(t *testing.T) {
		assert.True(t, IsFatalError(fmt.Errorf("credit balance is too low")))
		assert.True(t, IsFatalError(fmt.Errorf("overloaded")))
		assert.True(t, IsFatalError(fmt.Errorf("rate_limit_error")))
	})

	t.Run("google patterns are fatal", func(t *testing.T) {
		assert.True(t, IsFatalError(fmt.Errorf("RESOURCE_EXHAUSTED")))
		assert.True(t, IsFatalError(fmt.Errorf("quota exceeded")))
	})

	t.Run("deepseek patterns are fatal", func(t *testing.T) {
		assert.True(t, IsFatalError(fmt.Errorf("insufficient balance")))
		assert.True(t, IsFatalError(fmt.Errorf("rate limit reached")))
	})

	t.Run("non-fatal errors return false", func(t *testing.T) {
		assert.False(t, IsFatalError(fmt.Errorf("connection refused")))
		assert.False(t, IsFatalError(fmt.Errorf("timeout")))
		assert.False(t, IsFatalError(fmt.Errorf("invalid input")))
	})
}

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
