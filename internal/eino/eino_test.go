package eino

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
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

func TestTokenTracker(t *testing.T) {
	tracker := &TokenTracker{}

	tracker.Track(10, 20)
	tracker.Track(30, 40)

	tracker.mu.Lock()
	assert.Equal(t, 40, tracker.inputTokens)
	assert.Equal(t, 60, tracker.outputTokens)
	tracker.mu.Unlock()
}

func TestTokenTrackerPrintStats(t *testing.T) {
	tracker := &TokenTracker{}
	tracker.Track(100, 200)
	tracker.PrintStats()
}

func TestNewHandler(t *testing.T) {
	tracker := &TokenTracker{}
	handler := NewHandler(tracker)
	require.NotNil(t, handler)

	ctx := context.Background()
	info := &callbacks.RunInfo{Name: "test"}
	output := &model.CallbackOutput{
		Message: &schema.Message{
			ResponseMeta: &schema.ResponseMeta{
				Usage: &schema.TokenUsage{
					PromptTokens:     50,
					CompletionTokens: 100,
				},
			},
		},
	}

	handler.OnEnd(ctx, info, output)

	tracker.mu.Lock()
	assert.Equal(t, 50, tracker.inputTokens)
	assert.Equal(t, 100, tracker.outputTokens)
	tracker.mu.Unlock()
}

func TestNewHandlerNilResponseMeta(t *testing.T) {
	tracker := &TokenTracker{}
	handler := NewHandler(tracker)

	ctx := context.Background()
	info := &callbacks.RunInfo{Name: "test"}
	output := &model.CallbackOutput{
		Message: &schema.Message{},
	}

	handler.OnEnd(ctx, info, output)

	tracker.mu.Lock()
	assert.Equal(t, 0, tracker.inputTokens)
	assert.Equal(t, 0, tracker.outputTokens)
	tracker.mu.Unlock()
}

func TestNewHandlerNilMessage(t *testing.T) {
	tracker := &TokenTracker{}
	handler := NewHandler(tracker)

	ctx := context.Background()
	info := &callbacks.RunInfo{Name: "test"}
	output := &model.CallbackOutput{
		Message: nil,
	}

	handler.OnEnd(ctx, info, output)

	tracker.mu.Lock()
	assert.Equal(t, 0, tracker.inputTokens)
	assert.Equal(t, 0, tracker.outputTokens)
	tracker.mu.Unlock()
}

func TestNewHandlerNonModelOutput(t *testing.T) {
	tracker := &TokenTracker{}
	handler := NewHandler(tracker)

	ctx := context.Background()
	info := &callbacks.RunInfo{Name: "test"}
	output := "not a model callback output"

	handler.OnEnd(ctx, info, output)

	tracker.mu.Lock()
	assert.Equal(t, 0, tracker.inputTokens)
	assert.Equal(t, 0, tracker.outputTokens)
	tracker.mu.Unlock()
}
