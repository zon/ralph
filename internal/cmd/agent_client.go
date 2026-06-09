package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type AgentClient struct {
	ctx *context.Context
	oc  opencode.OCClient
}

func NewAgentClient(ctx *context.Context, oc opencode.OCClient) *AgentClient {
	return &AgentClient{ctx: ctx, oc: oc}
}

func (a *AgentClient) RunPicker(proj *project.Project) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	commitLog, err := getCommitLog(a.ctx, cfg.DefaultBranch)
	if err != nil {
		commitLog = ""
	}

	pickedReqPath := filepath.Join(filepath.Dir(proj.Path), "picked-requirement.yaml")

	setup := &project.IterationSetup{
		Project:       proj,
		CommitLog:     commitLog,
		PickedReqPath: pickedReqPath,
	}

	return project.PickRequirement(a.ctx, a.oc, setup)
}

func (a *AgentClient) RunDeveloper(proj *project.Project, req string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	commitLog, err := getCommitLog(a.ctx, cfg.DefaultBranch)
	if err != nil {
		commitLog = ""
	}

	setup := &project.IterationSetup{
		Project:   proj,
		CommitLog: commitLog,
		Config:    cfg,
	}

	return project.DevelopRequirement(a.ctx, a.oc, setup, req)
}

func (a *AgentClient) IsFatal(err error) bool {
	return opencode.IsFatalError(err)
}

func (a *AgentClient) GenerateChangelog(proj *project.Project) error {
	return ai.GenerateChangelog(a.ctx, a.oc)
}

func (a *AgentClient) FixServiceStartup(cfg *config.RalphConfig, err error) error {
	svcMgr := services.NewManager(a.ctx.Output())
	if failedSvc, startErr := svcMgr.Start(cfg.Services); startErr != nil {
		fixPrompt, buildErr := ai.BuildFixServicePrompt(a.ctx, failedSvc, startErr)
		if buildErr != nil {
			return buildErr
		}
		return ai.RunAgent(a.ctx, a.oc, fixPrompt)
	}
	return nil
}

func (a *AgentClient) WriteOrchestration(input *project.InputFile) error {
	prompt, err := ai.BuildWriteProjectPrompt(ai.WriteProjectPromptData{
		InputPath: input.Path(),
		InputType: "specification file",
	})
	if err != nil {
		return fmt.Errorf("failed to build write orchestration prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	return ai.RunAgent(a.ctx, a.oc, prompt)
}

func (a *AgentClient) WriteProject(input *project.InputFile) (*project.Project, error) {
	inputType := "orchestration file"
	var orchestrationPath string
	if input.IsSpec() {
		inputType = "specification file"
		orchestrationPath = filepath.Join(filepath.Dir(input.Path()), "orchestration.md")
	}

	prompt, err := ai.BuildWriteProjectPrompt(ai.WriteProjectPromptData{
		InputPath:        input.Path(),
		InputType:        inputType,
		HasOrchestration: input.IsSpec(),
		OrchestrationPath: orchestrationPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build write project prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	if err := ai.RunAgent(a.ctx, a.oc, prompt); err != nil {
		return nil, err
	}

	proj, err := findNewestProject()
	if err != nil {
		return nil, err
	}

	return proj, nil
}

func findNewestProject() (*project.Project, error) {
	entries, err := os.ReadDir("projects")
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var newestPath string
	var newestModTime int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime().UnixNano()
		if modTime > newestModTime {
			newestModTime = modTime
			newestPath = filepath.Join("projects", e.Name())
		}
	}

	if newestPath == "" {
		return nil, fmt.Errorf("no project file found in projects/ directory")
	}

	proj, err := project.LoadProject(newestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load generated project file: %w", err)
	}

	return proj, nil
}

func (a *AgentClient) PrintStats() {
	stats, err := a.oc.GetStats()
	if err != nil {
		return
	}
	a.ctx.Output().Infof("Input tokens: %s, Output tokens: %s, Cost: $%.2f", formatTokens(stats.InputTokens), formatTokens(stats.OutputTokens), stats.Cost)
}

func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func getCommitLog(ctx *context.Context, defaultBranch string) (string, error) {
	baseBranch := defaultBranch
	if ctx.BaseBranch() != "" {
		baseBranch = ctx.BaseBranch()
	}
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return "", err
	}
	if currentBranch == baseBranch {
		return "", nil
	}
	return git.GetCommitLog(baseBranch, 10)
}
