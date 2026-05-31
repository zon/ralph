package eino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"google.golang.org/genai"

	"github.com/zon/ralph/internal/auth"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

var validProviders = map[string]bool{
	"anthropic": true,
	"google":    true,
	"deepseek":  true,
}

func parseModel(model string) (provider, modelID string, err error) {
	idx := strings.Index(model, "/")
	if idx == -1 {
		return "", "", fmt.Errorf("invalid model string %q: no \"/\" found", model)
	}
	provider = model[:idx]
	modelID = model[idx+1:]
	if !validProviders[provider] {
		return "", "", fmt.Errorf("unknown provider %q: must be one of anthropic, google, deepseek", provider)
	}
	return provider, modelID, nil
}

func Complete(ctx context.Context, modelStr, prompt string) (string, error) {
	provider, modelID, err := parseModel(modelStr)
	if err != nil {
		return "", err
	}

	rootDir, err := git.RepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find repo root: %w", err)
	}

	keys, err := auth.Load(rootDir)
	if err != nil {
		return "", fmt.Errorf("failed to load auth keys: %w", err)
	}

	apiKey, ok := keys[provider]
	if !ok || apiKey == "" {
		return "", fmt.Errorf("API key for %q not found in auth.yaml", provider)
	}

	msg := schema.UserMessage(prompt)

	switch provider {
	case "anthropic":
		cm, err := claude.NewChatModel(ctx, &claude.Config{
			APIKey:    apiKey,
			Model:     modelID,
			MaxTokens: 4096,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create claude model: %w", err)
		}
		result, err := cm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return "", fmt.Errorf("model generation failed: %w", err)
		}
		return result.Content, nil

	case "google":
		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
		if err != nil {
			return "", fmt.Errorf("failed to create gemini client: %w", err)
		}
		cm, err := gemini.NewChatModel(ctx, &gemini.Config{
			Client: client,
			Model:  modelID,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create gemini model: %w", err)
		}
		result, err := cm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return "", fmt.Errorf("model generation failed: %w", err)
		}
		return result.Content, nil

	case "deepseek":
		cm, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: apiKey,
			Model:  modelID,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create deepseek model: %w", err)
		}
		result, err := cm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return "", fmt.Errorf("model generation failed: %w", err)
		}
		return result.Content, nil
	}

	return "", fmt.Errorf("unexpected provider %q", provider)
}

// RunAgent creates an agentic ReAct loop with the given model, variant, and prompt.
func RunAgent(ctx context.Context, modelStr, variant, prompt string, tracker *TokenTracker) error {
	provider, modelID, err := parseModel(modelStr)
	if err != nil {
		return err
	}

	rootDir, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("failed to find repo root: %w", err)
	}

	keys, err := auth.Load(rootDir)
	if err != nil {
		return fmt.Errorf("failed to load auth keys: %w", err)
	}

	apiKey, ok := keys[provider]
	if !ok || apiKey == "" {
		return fmt.Errorf("API key for %q not found in auth.yaml", provider)
	}

	var chatModel model.ToolCallingChatModel

	switch provider {
	case "anthropic":
		cfg := &claude.Config{
			APIKey:    apiKey,
			Model:     modelID,
			MaxTokens: 4096,
		}
		if variant != "" {
			budget := 2048
			if n, err := strconv.Atoi(variant); err == nil && n > 0 {
				budget = n
			}
			cfg.Thinking = &claude.Thinking{
				Enable:       true,
				BudgetTokens: budget,
			}
		}
		cm, err := claude.NewChatModel(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to create claude model: %w", err)
		}
		chatModel = cm

	case "google":
		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
		if err != nil {
			return fmt.Errorf("failed to create gemini client: %w", err)
		}
		cfg := &gemini.Config{
			Client: client,
			Model:  modelID,
		}
		cm, err := gemini.NewChatModel(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to create gemini model: %w", err)
		}
		chatModel = cm

	case "deepseek":
		cfg := &deepseek.ChatModelConfig{
			APIKey: apiKey,
			Model:  modelID,
		}
		cm, err := deepseek.NewChatModel(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to create deepseek model: %w", err)
		}
		chatModel = cm
	}

	callbacks.AppendGlobalHandlers(StreamingHandler(), NewHandler(tracker))

	msg := schema.UserMessage(prompt)

	toolOpts, err := react.WithTools(ctx, CodingTools()...)
	if err != nil {
		return fmt.Errorf("failed to configure tools: %w", err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	_, err = agent.Generate(ctx, []*schema.Message{msg}, toolOpts...)
	return err
}

// TokenTracker accumulates token usage across agent calls with thread-safe counters.
type TokenTracker struct {
	mu           sync.Mutex
	inputTokens  int
	outputTokens int
}

// Track adds the given token counts to the running totals.
func (t *TokenTracker) Track(inputTokens, outputTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inputTokens += inputTokens
	t.outputTokens += outputTokens
}

// PrintStats logs the accumulated token totals using the application logger.
func (t *TokenTracker) PrintStats() {
	t.mu.Lock()
	input := t.inputTokens
	output := t.outputTokens
	t.mu.Unlock()
	logger.Infof("Token usage — input: %d, output: %d, total: %d", input, output, input+output)
}

// NewHandler creates an eino callback handler that tracks token usage from model responses.
func NewHandler(tracker *TokenTracker) callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if cbOutput := model.ConvCallbackOutput(output); cbOutput != nil {
				if msg := cbOutput.Message; msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
					tracker.Track(msg.ResponseMeta.Usage.PromptTokens, msg.ResponseMeta.Usage.CompletionTokens)
				}
			}
			return ctx
		}).
		Build()
}

// StreamingHandler creates an eino callback handler that streams model output to stdout.
func StreamingHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			return ctx
		}).
		OnStartWithStreamInputFn(func(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
			return ctx
		}).
		OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			defer output.Close()
			for {
				chunk, err := output.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					break
				}
				if cbOutput := model.ConvCallbackOutput(chunk); cbOutput != nil {
					if msg := cbOutput.Message; msg != nil {
						if msg.Content != "" {
							fmt.Print(msg.Content)
						}
						for _, tc := range msg.ToolCalls {
							firstArg := extractFirstArg(tc.Function.Arguments)
							fmt.Printf("\n%s %s", tc.Function.Name, firstArg)
						}
					}
				}
			}
			return ctx
		}).
		Build()
}

func extractFirstArg(argsJSON string) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	for _, v := range args {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
