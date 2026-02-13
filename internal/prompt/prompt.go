package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// BuildDevelopPrompt creates a prompt for the AI agent to work on project requirements
// It includes recent git history, project requirements, and development instructions
func BuildDevelopPrompt(ctx *context.Context, projectFile string) (string, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would build development prompt")
		return "dry-run-prompt", nil
	}

	var builder strings.Builder

	// Header - matches develop.sh format exactly
	builder.WriteString("# Development Agent Context\n")
	builder.WriteString("\n")
	builder.WriteString("## Project Information\n")
	builder.WriteString("\n")
	builder.WriteString("You are an AI coding agent working on this project.\n")
	builder.WriteString("Your task is to implement requirements from the project file below.\n")
	builder.WriteString("\n")

	// Recent Git History - matches develop.sh format exactly
	builder.WriteString("## Recent Git History\n")
	builder.WriteString("\n")
	commits, err := git.GetRecentCommits(ctx, 20)
	if err != nil {
		logger.Warningf("Failed to get recent commits: %v", err)
		builder.WriteString("(Unable to retrieve git history)\n")
	} else if len(commits) == 0 {
		builder.WriteString("(No commits yet)\n")
	} else {
		for _, commit := range commits {
			builder.WriteString(commit)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")

	// Project Requirements - matches develop.sh format exactly
	builder.WriteString("## Project Requirements\n")
	builder.WriteString("\n")
	projectContent, err := os.ReadFile(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to read project file: %w", err)
	}
	builder.Write(projectContent)
	builder.WriteString("\n")

	// Development Instructions - loaded from config
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	builder.WriteString(ralphConfig.Instructions)
	builder.WriteString("\n")

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Infof("Generated prompt (%d bytes)", len(prompt))
	}

	return prompt, nil
}
