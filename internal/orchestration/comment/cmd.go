package comment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

type AIClient interface {
	RunAgent(prompt string) error
}

type ServicesClient interface {
	Start(services []config.Service) error
	Stop()
}

type CommentFlags struct {
	Body       string
	Repo       string
	Branch     string
	PR         string
	NoServices bool
	Verbose    bool
}

type CommentCmd struct {
	ai       AIClient
	services ServicesClient
}

func NewCommentCmd(ai AIClient, services ServicesClient) *CommentCmd {
	return &CommentCmd{
		ai:       ai,
		services: services,
	}
}

func (c *CommentCmd) Run(flags CommentFlags) error {
	if flags.Verbose {
		logger.SetVerbose(true)
	}

	projectFile := projectFileFromBranch(flags.Branch)
	absProjectFile, err := filepath.Abs(projectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absProjectFile)
	}

	if _, err := project.LoadProject(absProjectFile); err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	svcCleanup := c.startServicesIfNeeded(flags.NoServices, cfg.Services)
	if svcCleanup != nil {
		defer svcCleanup()
	}

	agentPrompt := renderInstructions(cfg.CommentInstructions, flags.Repo, flags.Branch, flags.Body, flags.PR)

	logger.Verbose("Running AI agent...")
	if err := c.ai.RunAgent(agentPrompt); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	return nil
}

func (c *CommentCmd) startServicesIfNeeded(noServices bool, services []config.Service) func() {
	if noServices || len(services) == 0 {
		return nil
	}
	if err := c.services.Start(services); err != nil {
		return nil
	}
	return c.services.Stop
}

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

func projectFileFromBranch(branch string) string {
	projectName := branch
	if strings.HasPrefix(branch, "ralph/") {
		projectName = strings.TrimPrefix(branch, "ralph/")
	} else {
		projectName = strings.ReplaceAll(branch, "/", "-")
	}
	return filepath.Join("projects", projectName+".yaml")
}
