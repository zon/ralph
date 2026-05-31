package eino

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
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

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func TestStreamingHandler_WritesTextToStdout(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{Content: "Hello, "},
		}, nil)
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{Content: "world!"},
		}, nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.Equal(t, "Hello, world!", output)
}

func TestStreamingHandler_WritesToolCallToStdout(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{
				ToolCalls: []schema.ToolCall{
					{
						Function: schema.FunctionCall{
							Name:      "read",
							Arguments: `{"path": "internal/eino/eino.go"}`,
						},
					},
				},
			},
		}, nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.Contains(t, output, "read internal/eino/eino.go")
}

func TestStreamingHandler_ClosesStream(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{Content: "done"},
		}, nil)
		sw.Close()
	}()

	handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)

	assert.Panics(t, func() {
		sr.Close()
	})
}

func TestStreamingHandler_OtherMethodsProduceNoOutput(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	ctx := context.Background()
	info := &callbacks.RunInfo{Name: "test"}

	handler.OnStart(ctx, info, "input")
	handler.OnEnd(ctx, info, "output")
	handler.OnError(ctx, info, assert.AnError)
	handler.OnStartWithStreamInput(ctx, info, nil)
}

func TestStreamingHandler_ToolCallFormat(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{
				ToolCalls: []schema.ToolCall{
					{
						Function: schema.FunctionCall{
							Name:      "bash",
							Arguments: `{"command": "ls -la"}`,
						},
					},
					{
						Function: schema.FunctionCall{
							Name:      "write",
							Arguments: `{"path": "test.txt", "content": "hello"}`,
						},
					},
				},
			},
		}, nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.Contains(t, output, "bash ls -la")
	assert.Contains(t, output, "write test.txt")
}

func TestStreamingHandler_ContentAndToolCalls(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{Content: "Let me check that file."},
		}, nil)
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{
				ToolCalls: []schema.ToolCall{
					{
						Function: schema.FunctionCall{
							Name:      "read",
							Arguments: `{"path": "test.txt"}`,
						},
					},
				},
			},
		}, nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.True(t, strings.Contains(output, "Let me check that file."), "output should contain text content")
	assert.True(t, strings.Contains(output, "read test.txt"), "output should contain tool call line")
}

func TestStreamingHandler_NonModelOutput(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send("not a model output", nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.Empty(t, output)
}

func TestStreamingHandler_EmptyContent(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send(&model.CallbackOutput{
			Message: &schema.Message{Content: ""},
		}, nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.Empty(t, output)
}

func TestStreamingHandler_NilMessage(t *testing.T) {
	handler := StreamingHandler()
	require.NotNil(t, handler)

	sr, sw := schema.Pipe[callbacks.CallbackOutput](10)
	go func() {
		defer sw.Close()
		sw.Send(&model.CallbackOutput{
			Message: nil,
		}, nil)
	}()

	output := captureStdout(func() {
		handler.OnEndWithStreamOutput(context.Background(), &callbacks.RunInfo{}, sr)
	})

	assert.Empty(t, output)
}
