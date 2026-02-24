package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/services"
)

// CommentCmd is the command for running a comment-triggered development iteration
type CommentCmd struct {
	Body    string `arg:"" help:"Comment body text"`
	Repo    string `help:"Repository in owner/repo format, e.g. zon/ralph" required:""`
	Branch  string `help:"PR branch name" required:""`
	PR      string `help:"Pull request number" required:""`
	DryRun   bool `help:"Simulate execution without making changes" default:"false"`
	Verbose  bool `help:"Enable verbose logging" default:"false"`
	NoNotify bool `help:"Disable desktop notifications" default:"false" hidden:""`

	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the comment command (implements kong.Run interface)
func (c *CommentCmd) Run() error {
	if c.Verbose {
		logger.SetVerbose(true)
	}

	if c.DryRun {
		logger.Verbose("=== DRY-RUN MODE: No changes will be made ===")
	}

	// Derive and validate project file
	projectFile := projectFileFromBranch(c.Branch)
	absProjectFile, err := filepath.Abs(projectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absProjectFile)
	}

	// Load project
	if _, err := config.LoadProject(absProjectFile); err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	// Load ralph config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := &execcontext.Context{
		ProjectFile: projectFile,
		DryRun:      c.DryRun,
		Verbose:     c.Verbose,
		NoNotify:    c.NoNotify,
	}

	// Start services
	svcMgr := services.NewManager()
	if len(cfg.Services) > 0 {
		logger.Verbosef("Starting %d service(s)...", len(cfg.Services))
		if _, err := svcMgr.Start(cfg.Services, c.DryRun); err != nil {
			logger.Verbosef("Service startup failed: %v", err)
		} else {
			if c.cleanupRegistrar != nil {
				c.cleanupRegistrar(func() { svcMgr.Stop() })
			}
			defer func() {
				logger.Verbose("Stopping services...")
				svcMgr.Stop()
			}()
			logger.Verbose("All services started and healthy")
		}
	}

	// Generate comment prompt from rendered instructions template
	agentPrompt := renderInstructions(cfg.CommentInstructions, c.Repo, c.Branch, c.Body, c.PR)

	// Run agent
	logger.Verbose("Running AI agent...")
	if err := ai.RunAgent(ctx, agentPrompt); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	return nil
}

// renderInstructions renders a Go template with PR event context.
// tmplText is the template, repo is "owner/name", branch is the PR branch,
// body is the comment body (or empty for merge), pr is the PR number string.
func renderInstructions(tmplText, repo, branch, body, pr string) string {
	parts := strings.SplitN(repo, "/", 2)
	repoOwner, repoName := "", ""
	if len(parts) == 2 {
		repoOwner = parts[0]
		repoName = parts[1]
	}
	tmpl, err := template.New("instructions").Parse(tmplText)
	if err != nil {
		return tmplText
	}
	data := struct {
		CommentBody string
		PRNumber    string
		PRBranch    string
		RepoOwner   string
		RepoName    string
	}{
		CommentBody: body,
		PRNumber:    pr,
		PRBranch:    branch,
		RepoOwner:   repoOwner,
		RepoName:    repoName,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return tmplText
	}
	return buf.String()
}

// projectFileFromBranch derives the project file path from the PR head branch name.
// Branch "ralph/<project-name>" → "projects/<project-name>.yaml"
// Other branches: branch name with slashes replaced by dashes.
func projectFileFromBranch(branch string) string {
	projectName := branch
	if strings.HasPrefix(branch, "ralph/") {
		projectName = strings.TrimPrefix(branch, "ralph/")
	} else {
		projectName = strings.ReplaceAll(branch, "/", "-")
	}
	return filepath.Join("projects", projectName+".yaml")
}
