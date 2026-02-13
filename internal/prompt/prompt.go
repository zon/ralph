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
	var builder strings.Builder

	// Header - matches develop.sh format exactly
	builder.WriteString("# Development Agent Context\n")
	builder.WriteString("\n")
	builder.WriteString("## Project Information\n")
	builder.WriteString("\n")
	builder.WriteString("You are an AI coding agent working on this project.\n")
	builder.WriteString("Your task is to implement requirements from the project file below.\n")
	builder.WriteString("\n")

	// Load config once for both git history check and instructions
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Recent Git History - only include if current branch is not the base branch
	currentBranch, err := git.GetCurrentBranch(ctx)
	if err != nil {
		logger.Warningf("Failed to get current branch: %v", err)
	} else if currentBranch != ralphConfig.BaseBranch {
		// Show all commits in this branch (since base branch)
		commitLog, err := git.GetCommitLog(ctx, ralphConfig.BaseBranch)
		if err != nil {
			logger.Warningf("Failed to get branch commits: %v", err)
		} else if commitLog != "" {
			builder.WriteString("## Recent Git History\n")
			builder.WriteString("\n")
			builder.WriteString(commitLog)
			builder.WriteString("\n")
			builder.WriteString("\n")
		}
	}

	// Project Requirements - matches develop.sh format exactly
	builder.WriteString("## Project Requirements\n")
	builder.WriteString("\n")
	projectContent, err := os.ReadFile(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to read project file: %w", err)
	}
	builder.Write(projectContent)
	builder.WriteString("\n")

	// Development Instructions - already loaded from config above
	builder.WriteString(ralphConfig.Instructions)
	builder.WriteString("\n")

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Infof("Generated prompt (%d bytes)", len(prompt))
	}

	return prompt, nil
}
