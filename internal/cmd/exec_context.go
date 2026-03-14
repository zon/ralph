package cmd

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/context"
)

// createExecutionContext creates a new context with common environment variables applied.
// This is used by various commands to ensure consistent context initialization
// while keeping the internal/context package free of environment variable dependencies.
func createExecutionContext() *context.Context {
	ctx := &context.Context{}

	// Detection for workflow container environment
	if os.Getenv("RALPH_WORKFLOW_EXECUTION") == "true" {
		ctx.SetWorkflowExecution(true)
	}

	// Repository information
	ctx.SetRepoOwner(os.Getenv("GITHUB_REPO_OWNER"))
	ctx.SetRepoName(os.Getenv("GITHUB_REPO_NAME"))

	// Project and Branch settings
	if val := os.Getenv("PROJECT_PATH"); val != "" {
		ctx.SetProjectFile(val)
	}
	if val := os.Getenv("PROJECT_BRANCH"); val != "" {
		ctx.SetBranch(val)
	}
	if val := os.Getenv("BASE_BRANCH"); val != "" {
		ctx.SetBaseBranch(val)
	}

	// Execution settings
	if os.Getenv("RALPH_VERBOSE") == "true" {
		ctx.SetVerbose(true)
	}
	if os.Getenv("RALPH_NO_SERVICES") == "true" {
		ctx.SetNoServices(true)
	}
	if val := os.Getenv("RALPH_DEBUG_BRANCH"); val != "" {
		ctx.SetDebugBranch(val)
	}
	if val := os.Getenv("INSTRUCTIONS_MD"); val != "" {
		ctx.SetInstructionsMD(val)
	}

	// Iteration limits
	if val := os.Getenv("RALPH_MAX_ITERATIONS"); val != "" {
		var max int
		if _, err := fmt.Sscanf(val, "%d", &max); err == nil {
			ctx.SetMaxIterations(max)
		}
	}

	return ctx
}
