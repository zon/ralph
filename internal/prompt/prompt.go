package prompt

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// BuildServiceFixPrompt creates a prompt focused solely on fixing a failed service.
func BuildServiceFixPrompt(ctx *context.Context, svc config.Service, svcErr error) string {
	cmd := svc.Command
	if len(svc.Args) > 0 {
		cmd = fmt.Sprintf("%s %s", svc.Command, strings.Join(svc.Args, " "))
	}

	data := struct {
		Notes       []string
		ServiceName string
		ServiceCmd  string
		ServicePort int
		Error       string
	}{
		Notes:       ctx.Notes,
		ServiceName: svc.Name,
		ServiceCmd:  cmd,
		ServicePort: svc.Port,
		Error:       svcErr.Error(),
	}

	tmpl := template.Must(template.New("fix-service").Parse(config.DefaultFixServiceInstructions))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		logger.Warningf("Failed to execute fix-service template: %v", err)
		return ""
	}
	return buf.String()
}

// BuildDevelopPrompt creates a prompt for the AI agent to work on project requirements
// It includes recent git history, project requirements, and development instructions
func BuildDevelopPrompt(ctx *context.Context, projectFile string) (string, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	projectContent, err := os.ReadFile(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to read project file: %w", err)
	}

	promptTmpl := ralphConfig.Instructions
	if ctx.Instructions != "" {
		instructionsData, err := os.ReadFile(ctx.Instructions)
		if err != nil {
			return "", fmt.Errorf("failed to read instructions file %s: %w", ctx.Instructions, err)
		}
		promptTmpl = string(instructionsData)
	}

	var commitLog string
	currentBranch, err := git.GetCurrentBranch(ctx)
	if err != nil {
		logger.Warningf("Failed to get current branch: %v", err)
	} else if currentBranch != ralphConfig.BaseBranch {
		if log, err := git.GetCommitLog(ctx, ralphConfig.BaseBranch, 10); err != nil {
			logger.Warningf("Failed to get branch commits: %v", err)
		} else {
			commitLog = log
		}
	}

	data := struct {
		Notes          []string
		CommitLog      string
		ProjectContent string
		Services       []config.Service
	}{
		Notes:          ctx.Notes,
		CommitLog:      commitLog,
		ProjectContent: strings.TrimRight(string(projectContent), "\n"),
		Services:       ralphConfig.Services,
	}

	tmpl, err := template.New("develop-prompt").Parse(promptTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render prompt: %w", err)
	}

	prompt := buf.String()
	if ctx.IsVerbose() {
		logger.Infof("Generated prompt (%d bytes)", len(prompt))
	}
	return prompt, nil
}
