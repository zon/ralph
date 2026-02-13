package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/zon/ralph/internal/config"
	appcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// RunAgent executes an AI agent with the given prompt
// In dry-run mode, it logs what would be executed without actually calling the LLM
func RunAgent(ctx *appcontext.Context, prompt string) error {
	if ctx.IsDryRun() {
		// Truncate prompt for logging in dry-run mode
		truncated := prompt
		if len(prompt) > 200 {
			truncated = prompt[:200] + "..."
		}
		logger.Info(fmt.Sprintf("[DRY-RUN] Would run agent with prompt (first 200 chars): %s", truncated))
		return nil
	}

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load secrets
	secrets, err := config.LoadRalphSecrets()
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}

	// Determine provider and model
	provider := ralphConfig.LLMProvider
	if provider == "" {
		provider = "deepseek" // Default to deepseek
	}

	model := ralphConfig.LLMModel
	if model == "" {
		// Set provider-specific defaults
		switch provider {
		case "deepseek":
			model = "deepseek-reasoner" // R1 model for advanced reasoning
		default:
			return fmt.Errorf("no model specified for provider: %s", provider)
		}
	}

	// Get API key for the provider
	apiKey, ok := secrets.APIKeys[provider]
	if !ok || apiKey == "" {
		return fmt.Errorf("no API key found for provider '%s' in secrets", provider)
	}

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Using LLM provider: %s, model: %s", provider, model))
	}

	// Execute prompt based on provider
	logger.Info("Starting AI agent execution...")

	var response string
	switch provider {
	case "deepseek":
		response, err = callDeepSeekAPI(apiKey, model, prompt, ctx.IsVerbose())
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Stream response to stdout
	fmt.Println(response)

	logger.Success("Agent execution completed")

	return nil
}

// callDeepSeekAPI calls the DeepSeek Chat API using go-deepseek client
func callDeepSeekAPI(apiKey, model, prompt string, verbose bool) (string, error) {
	client, err := deepseek.NewClient(apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to create DeepSeek client: %w", err)
	}

	chatReq := &request.ChatCompletionsRequest{
		Model:  model,
		Stream: false,
		Messages: []*request.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	chatResp, err := client.CallChatCompletionsChat(context.Background(), chatReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	if verbose {
		logger.Info(fmt.Sprintf("Token usage: prompt=%d, completion=%d, total=%d",
			chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, chatResp.Usage.TotalTokens))
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GeneratePRSummary generates a pull request summary using AI
// It includes project description, status, commits, and diff
func GeneratePRSummary(ctx *appcontext.Context, projectFile string, iterations int) (string, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would generate PR summary")
		return "dry-run-pr-summary", nil
	}

	// Load project file
	project, err := config.LoadProject(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to load project: %w", err)
	}

	// Get completion status
	allComplete, passingCount, failingCount := config.CheckCompletion(project)

	var builder strings.Builder

	// Header
	builder.WriteString("# Pull Request Summary Generation\n\n")
	builder.WriteString("Please generate a concise 3-5 paragraph pull request summary for the following project work.\n\n")

	// Project Information
	builder.WriteString("## Project\n\n")
	builder.WriteString(fmt.Sprintf("**Name**: %s\n", project.Name))
	if project.Description != "" {
		builder.WriteString(fmt.Sprintf("**Description**: %s\n", project.Description))
	}
	builder.WriteString(fmt.Sprintf("**Iterations**: %d\n", iterations))
	builder.WriteString(fmt.Sprintf("**Status**: %d passing, %d failing (complete: %v)\n\n", passingCount, failingCount, allComplete))

	// Requirements
	builder.WriteString("## Requirements\n\n")
	for i, req := range project.Requirements {
		status := "❌"
		if req.Passing {
			status = "✅"
		}
		builder.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, req.Description))
		if req.Category != "" {
			builder.WriteString(fmt.Sprintf("   - Category: %s\n", req.Category))
		}
	}
	builder.WriteString("\n")

	// Get commits since main
	builder.WriteString("## Commits\n\n")
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	baseBranch := ralphConfig.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	commits, err := git.GetCommitsSince(ctx, baseBranch)
	if err != nil {
		logger.Warning(fmt.Sprintf("Failed to get commits: %v", err))
		builder.WriteString("(Unable to retrieve commits)\n\n")
	} else if len(commits) == 0 {
		builder.WriteString("(No commits)\n\n")
	} else {
		for _, commit := range commits {
			builder.WriteString(commit)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Get diff since main
	builder.WriteString("## Changes (Diff)\n\n")
	diff, err := git.GetDiffSince(ctx, baseBranch)
	if err != nil {
		logger.Warning(fmt.Sprintf("Failed to get diff: %v", err))
		builder.WriteString("(Unable to retrieve diff)\n\n")
	} else if diff == "" {
		builder.WriteString("(No changes)\n\n")
	} else {
		builder.WriteString("```diff\n")
		builder.WriteString(diff)
		builder.WriteString("\n```\n\n")
	}

	// Instructions
	builder.WriteString("## Instructions\n\n")
	builder.WriteString("Generate a concise 3-5 paragraph summary suitable for a GitHub pull request description.\n")
	builder.WriteString("Focus on:\n")
	builder.WriteString("- What was accomplished\n")
	builder.WriteString("- Key changes made\n")
	builder.WriteString("- Any notable implementation details\n")
	builder.WriteString("- Current status (all requirements passing or work in progress)\n\n")
	builder.WriteString("Do NOT include:\n")
	builder.WriteString("- Code snippets\n")
	builder.WriteString("- Detailed line-by-line explanations\n")
	builder.WriteString("- Test output\n\n")
	builder.WriteString("Return ONLY the summary text, no additional commentary.\n")

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Generated PR summary prompt (%d bytes)", len(prompt)))
	}

	// Load configuration and secrets
	secrets, err := config.LoadRalphSecrets()
	if err != nil {
		return "", fmt.Errorf("failed to load secrets: %w", err)
	}

	// Determine provider and model
	provider := ralphConfig.LLMProvider
	if provider == "" {
		provider = "deepseek"
	}

	model := ralphConfig.LLMModel
	if model == "" {
		switch provider {
		case "deepseek":
			model = "deepseek-reasoner" // R1 model for advanced reasoning
		default:
			return "", fmt.Errorf("no model specified for provider: %s", provider)
		}
	}

	// Get API key
	apiKey, ok := secrets.APIKeys[provider]
	if !ok || apiKey == "" {
		return "", fmt.Errorf("no API key found for provider '%s' in secrets", provider)
	}

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Generating PR summary using %s/%s", provider, model))
	}

	// Generate summary
	logger.Info("Generating PR summary...")

	var summary string
	switch provider {
	case "deepseek":
		summary, err = callDeepSeekAPI(apiKey, model, prompt, ctx.IsVerbose())
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	summary = strings.TrimSpace(summary)

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Generated summary (%d bytes)", len(summary)))
	}

	logger.Success("PR summary generated")

	return summary, nil
}

// ValidateConfig checks if AI configuration is valid
func ValidateConfig(ralphConfig *config.RalphConfig, secrets *config.RalphSecrets) error {
	provider := ralphConfig.LLMProvider
	if provider == "" {
		provider = "deepseek" // Default
	}

	apiKey, ok := secrets.APIKeys[provider]
	if !ok || apiKey == "" {
		return fmt.Errorf("no API key found for provider '%s'", provider)
	}

	return nil
}
