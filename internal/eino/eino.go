package eino

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/schema"
	"google.golang.org/genai"

	"github.com/zon/ralph/internal/auth"
	"github.com/zon/ralph/internal/git"
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
