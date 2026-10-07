package cmd

import (
	"errors"
	"fmt"
	"os"
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

// maxPickerAttempts is how many times the picker prompt is run before the
// command reports an error naming the limit.
const maxPickerAttempts = 3

// RunPicker asks the AI to select one incomplete item and returns it. A
// selection is usable only when picked-item.txt holds the text of one of the
// incomplete items; when a run leaves no usable selection — the file missing,
// empty, or naming no incomplete item — the same prompt is re-run until
// maxPickerAttempts attempts have been made. When every attempt is unusable the
// returned error names the attempt limit. An opencode execution failure is
// returned immediately and is never retried.
func (a *AgentClient) RunPicker(proj *project.Project, incomplete []project.Item, previousErr error) (project.Item, error) {
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
		PreviousError:  previousErr,
	})
	if err != nil {
		return project.Item{}, fmt.Errorf("failed to build pick prompt: %w", err)
	}

	if a.ctx.IsVerbose() {
		a.ctx.Output().Debug(prompt)
	}

	var lastErr error
	for attempt := 1; attempt <= maxPickerAttempts; attempt++ {
		if err := ai.RunAgentPrimary(a.ctx, a.oc, prompt); err != nil {
			return project.Item{}, err
		}

		text, err := readPickedItem()
		if err == nil {
			item, ok := findItemByText(incomplete, text)
			if !ok {
				err = fmt.Errorf("picker wrote text that names no incomplete item: %q", text)
			} else {
				return item, nil
			}
		}
		lastErr = err
	}
	return project.Item{}, fmt.Errorf("no usable selection after the %d-attempt limit: %w", maxPickerAttempts, lastErr)
}

func (a *AgentClient) RunDeveloper(proj *project.Project, item project.Item, previousErr error) error {
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
		ItemValue:       item.Text(),
		Trailer:         trailer.Format(proj.Slug, item.Hash()),
		ProjectFilePath: proj.Path,
		Services:        cfg.Services,
		Instructions:    cfg.Instructions,
		PreviousError:   previousErr,
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

// renderItems renders each incomplete item with its key, when it has one, above
// the item's exact text, so the picker can choose one and echo its text.
func renderItems(items []project.Item) string {
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		if key := it.Key(); key != "" {
			fmt.Fprintf(&b, "**%s**\n\n", key)
		}
		fmt.Fprintf(&b, "%s\n", it.Text())
	}
	return strings.TrimRight(b.String(), "\n")
}

// findItemByText returns the incomplete item whose completion hash matches the
// picked text. Matching by the item's own hash, rather than by a position,
// keeps the selection tied to the item's identity however the array is ordered.
func findItemByText(incomplete []project.Item, text string) (project.Item, bool) {
	hash := trailer.Hash(text)
	for _, it := range incomplete {
		if it.Hash() == hash {
			return it, true
		}
	}
	return project.Item{}, false
}

// readPickedItem reads the item text the picker agent wrote to
// picked-item.txt.
func readPickedItem() (string, error) {
	data, err := os.ReadFile("picked-item.txt")
	if err != nil {
		return "", fmt.Errorf("failed to read picked item: %w", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", errors.New("picked item is empty")
	}
	if err := os.Remove("picked-item.txt"); err != nil {
		return "", fmt.Errorf("failed to remove picked-item.txt: %w", err)
	}
	return text, nil
}

func (a *AgentClient) IsFatal(err error) bool {
	return opencode.IsFatalError(err)
}

func (a *AgentClient) GenerateChangelog(proj *project.Project) error {
	return ai.GenerateChangelog(a.ctx, a.oc)
}

// ResolveMergeConflicts invokes the configured agent to resolve a base-branch
// merge conflict, run the tests, and stage the resolved files. It runs with the
// configured agent because resolving conflicts writes repository code.
func (a *AgentClient) ResolveMergeConflicts(baseBranch, projectBranch string) error {
	prompt, err := ai.BuildResolveMergeConflictsPrompt(baseBranch, projectBranch)
	if err != nil {
		return err
	}
	return ai.RunAgent(a.ctx, a.oc, prompt)
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

func (a *AgentClient) PrintStats() {
	stats, err := a.oc.GetStats()
	if err != nil {
		return
	}
	a.ctx.Output().Infof("%s", stats.Formatted())
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
