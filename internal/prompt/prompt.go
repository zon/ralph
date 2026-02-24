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

// BuildServiceFixPrompt creates a minimal prompt focused solely on fixing a failed service.
// It omits the project file, git history, and development instructions.
func BuildServiceFixPrompt(ctx *context.Context, svc config.Service) string {
	var builder strings.Builder

	for _, note := range ctx.Notes {
		builder.WriteString(note)
		builder.WriteString("\n")
	}

	builder.WriteString("\n## Service Details\n\n")
	cmd := svc.Command
	if len(svc.Args) > 0 {
		cmd = fmt.Sprintf("%s %s", svc.Command, strings.Join(svc.Args, " "))
	}
	builder.WriteString(fmt.Sprintf("- **Name:** %s\n", svc.Name))
	builder.WriteString(fmt.Sprintf("- **Start command:** `%s`\n", cmd))
	if svc.Port > 0 {
		builder.WriteString(fmt.Sprintf("- **Health check:** port %d must be accepting connections\n", svc.Port))
	}

	return builder.String()
}

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
	builder.WriteString("\n")

	// Include any runtime notes from context
	if ctx.HasNotes() {
		builder.WriteString("## System Notes\n")
		builder.WriteString("\n")
		for _, note := range ctx.Notes {
			builder.WriteString(note)
			builder.WriteString("\n\n")
		}
	}

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
		commitLog, err := git.GetCommitLog(ctx, ralphConfig.BaseBranch, 10)
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

	// Development Instructions - use --instructions file if provided, otherwise use config instructions
	instructions := ralphConfig.Instructions
	if ctx.Instructions != "" {
		instructionsData, err := os.ReadFile(ctx.Instructions)
		if err != nil {
			return "", fmt.Errorf("failed to read instructions file %s: %w", ctx.Instructions, err)
		}
		instructions = string(instructionsData)
	}
	builder.WriteString(instructions)
	builder.WriteString("\n")

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Infof("Generated prompt (%d bytes)", len(prompt))
	}

	return prompt, nil
}
