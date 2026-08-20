package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/trailer"
)

type AgentClient struct {
	ctx *context.Context
	oc  opencode.OCClient
}

func NewAgentClient(ctx *context.Context, oc opencode.OCClient) *AgentClient {
	return &AgentClient{ctx: ctx, oc: oc}
}

func (a *AgentClient) RunPicker(proj *project.Project, incomplete []project.Item) (project.Item, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return project.Item{}, fmt.Errorf("failed to load config: %w", err)
	}

	commitLog, err := getCommitLog(a.ctx, cfg.DefaultBranch)
	if err != nil {
		commitLog = ""
	}

	prompt, err := ai.BuildItemPickPrompt(ai.ItemPickPromptData{
		Notes:          a.ctx.Notes(),
		CommitLog:      commitLog,
		ProjectContent: projectContent(proj),
		Items:          renderItems(incomplete),
	})
	if err != nil {
		return project.Item{}, fmt.Errorf("failed to build pick prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	if err := ai.RunAgentPrimary(a.ctx, a.oc, prompt); err != nil {
		return project.Item{}, err
	}

	idx, err := readPickedIndex()
	if err != nil {
		return project.Item{}, err
	}
	if idx < 0 || idx >= len(proj.Items) {
		return project.Item{}, fmt.Errorf("picker reported index %d which is outside the resolved item array (%d items)", idx, len(proj.Items))
	}
	return proj.Items[idx], nil
}

func (a *AgentClient) RunDeveloper(proj *project.Project, item project.Item) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	commitLog, err := getCommitLog(a.ctx, cfg.DefaultBranch)
	if err != nil {
		commitLog = ""
	}

	prompt, err := ai.BuildItemDevelopPrompt(ai.ItemDevelopPromptData{
		Notes:           a.ctx.Notes(),
		CommitLog:       commitLog,
		ProjectContent:  projectContent(proj),
		ItemIndex:       item.Index,
		ItemKey:         item.Key(),
		ItemValue:       renderItemValue(item.Value),
		Trailer:         trailer.Format(item.Index, item.Key()),
		ProjectFilePath: proj.Path,
		Services:        cfg.Services,
		Instructions:    cfg.Instructions,
	})
	if err != nil {
		return fmt.Errorf("failed to build development prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	return ai.RunAgent(a.ctx, a.oc, prompt)
}

// projectContent returns the project file's raw content, falling back to a
// YAML rendering when the raw document was not retained.
func projectContent(proj *project.Project) string {
	if proj.Doc != nil && proj.Doc.Raw != "" {
		return proj.Doc.Raw
	}
	data, err := yaml.Marshal(proj)
	if err != nil {
		return ""
	}
	return string(data)
}

// renderItems renders each incomplete item with its index and key so the picker
// can select one of them.
func renderItems(items []project.Item) string {
	var b strings.Builder
	for _, it := range items {
		if key := it.Key(); key != "" {
			fmt.Fprintf(&b, "item %d (%s):\n%s\n", it.Index, key, renderItemValue(it.Value))
		} else {
			fmt.Fprintf(&b, "item %d:\n%s\n", it.Index, renderItemValue(it.Value))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderItemValue renders an item's raw resolved value verbatim as YAML.
func renderItemValue(v any) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return strings.TrimRight(string(data), "\n")
}

// readPickedIndex reads the 0-based index the picker agent wrote to
// picked-item-index.txt.
func readPickedIndex() (int, error) {
	data, err := os.ReadFile("picked-item-index.txt")
	if err != nil {
		return 0, fmt.Errorf("failed to read picked item index: %w", err)
	}
	idx, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("picked item index is not an integer: %q", strings.TrimSpace(string(data)))
	}
	if err := os.Remove("picked-item-index.txt"); err != nil {
		return 0, fmt.Errorf("failed to remove picked-item-index.txt: %w", err)
	}
	return idx, nil
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
	prompt, err := ai.BuildWriteOrchestrationPrompt(ai.WriteOrchestrationPromptData{
		SpecPath: input.Path(),
	})
	if err != nil {
		return fmt.Errorf("failed to build write orchestration prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	return ai.RunAgentPrimary(a.ctx, a.oc, prompt)
}

func (a *AgentClient) WriteProject(input *project.InputFile) (string, error) {
	inputType := "orchestration file"
	var orchestrationPath string
	if input.IsSpec() {
		inputType = "specification file"
		orchestrationPath = filepath.Join(filepath.Dir(input.Path()), "orchestration.md")
	}

	prompt, err := ai.BuildWriteProjectPrompt(ai.WriteProjectPromptData{
		InputPath:         input.Path(),
		InputType:         inputType,
		HasOrchestration:  input.IsSpec(),
		OrchestrationPath: orchestrationPath,
	})
	if err != nil {
		return "", fmt.Errorf("failed to build write project prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	if err := ai.RunAgentPrimary(a.ctx, a.oc, prompt); err != nil {
		return "", err
	}

	path, err := findNewestProjectPath()
	if err != nil {
		return "", err
	}

	return path, nil
}

func findNewestProjectPath() (string, error) {
	entries, err := os.ReadDir("projects")
	if err != nil {
		return "", fmt.Errorf("failed to read projects directory: %w", err)
	}

	var newestPath string
	var newestModTime int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
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
		return "", fmt.Errorf("no project file found in projects/ directory")
	}

	return newestPath, nil
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
