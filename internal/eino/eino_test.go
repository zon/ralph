package eino

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModel(t *testing.T) {
	t.Run("anthropic/claude-sonnet-4-6", func(t *testing.T) {
		provider, modelID, err := parseModel("anthropic/claude-sonnet-4-6")
		require.NoError(t, err)
		assert.Equal(t, "anthropic", provider)
		assert.Equal(t, "claude-sonnet-4-6", modelID)
	})

	t.Run("google/gemini-2.5-pro", func(t *testing.T) {
		provider, modelID, err := parseModel("google/gemini-2.5-pro")
		require.NoError(t, err)
		assert.Equal(t, "google", provider)
		assert.Equal(t, "gemini-2.5-pro", modelID)
	})

	t.Run("deepseek/deepseek-chat", func(t *testing.T) {
		provider, modelID, err := parseModel("deepseek/deepseek-chat")
		require.NoError(t, err)
		assert.Equal(t, "deepseek", provider)
		assert.Equal(t, "deepseek-chat", modelID)
	})

	t.Run("no slash returns error", func(t *testing.T) {
		_, _, err := parseModel("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no \"/\" found")
	})

	t.Run("unknown provider returns error", func(t *testing.T) {
		_, _, err := parseModel("openai/gpt-4")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}
