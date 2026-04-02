package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/run"
	"github.com/zon/ralph/internal/workflow"

	_ "embed"
)

//go:embed review-instructions.md
var reviewInstructions string

const ralphProjectDocURL = "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/projects.md"

type itemWithIndex struct {
	item config.ReviewItem
	idx  int
}

func shuffleComponents(components []OverviewComponent, seed int64) []OverviewComponent {
	if len(components) == 0 {
		return components
	}
	rng := rand.New(rand.NewSource(seed))
	shuffled := make([]OverviewComponent, len(components))
	copy(shuffled, components)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func shuffleItemsWithIndices(items []config.ReviewItem, seed int64) []itemWithIndex {
	if len(items) == 0 {
		return []itemWithIndex{}
	}
	rng := rand.New(rand.NewSource(seed))
	withIdx := make([]itemWithIndex, len(items))
	for i, item := range items {
		withIdx[i] = itemWithIndex{item: item, idx: i}
	}
	rng.Shuffle(len(withIdx), func(i, j int) {
		withIdx[i], withIdx[j] = withIdx[j], withIdx[i]
	})
	return withIdx
}

type ReviewCmd struct {
	ProjectFile string `help:"Path to output project YAML file" name:"project" short:"p"`
	Model       string `help:"Override the AI model from config" name:"model" optional:""`
	Base        string `help:"Override the base branch for PR creation" name:"base" optional:"" short:"B"`
	Local       bool   `help:"Run on this machine instead of submitting to Argo Workflows" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`
	Context     string `help:"Kubernetes context to use" name:"context" optional:""`
	Seed        int64  `help:"Random seed for shuffling components and review items (0 = random)" default:"0"`
	prSubmitted bool   // tracks whether a PR has already been submitted in this run
}

func (r *ReviewCmd) Run() error {
	if r.Verbose {
		logger.SetVerbose(true)
	}

	reviewName := "review-" + time.Now().Format("2006-01-02")

	if r.ProjectFile == "" {
		r.ProjectFile = "projects/" + reviewName + ".yaml"
	}

	absProjectFile, err := filepath.Abs(r.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	projectDir := filepath.Dir(absProjectFile)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	overviewPath, err := git.TmpPath("overview.json")
	if err != nil {
		return fmt.Errorf("failed to resolve overview path: %w", err)
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(ralphConfig.Review.Items) == 0 {
		return fmt.Errorf("no review items found in config")
	}

	projectDoc, err := fetchRalphProjectDoc()
	if err != nil {
		logger.Verbosef("Failed to fetch Ralph project doc: %v", err)
		projectDoc = ""
	}

	model := r.resolveModel(ralphConfig)

	ctx := createExecutionContext()
	ctx.SetProjectFile(absProjectFile)
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(model)
	ctx.SetLocal(r.Local)
	ctx.SetKubeContext(r.Context)

	startingBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	baseBranch := resolveBaseBranch(r.Base, startingBranch, reviewName, ralphConfig.DefaultBranch)
	ctx.SetBaseBranch(baseBranch)

	if !r.Local {
		return r.submitToArgo(ctx, startingBranch)
	}

	projectChanged := false

	overview, err := r.runOverview(ctx, overviewPath, absProjectFile, reviewName)
	if err != nil {
		return fmt.Errorf("overview step failed: %w", err)
	}

	projectChanged, err = r.runReview(ctx, overview, absProjectFile, projectDoc, reviewName, baseBranch, ralphConfig)
	if err != nil {
		return fmt.Errorf("review step failed: %w", err)
	}

	if projectChanged && !r.prSubmitted {
		if err := r.submitPR(ctx, absProjectFile, reviewName, baseBranch); err != nil {
			logger.Warningf("Failed to create pull request: %v", err)
		}
	}

	if err := r.printStats(); err != nil {
		logger.Verbosef("Failed to print stats: %v", err)
	}

	return nil
}

func (r *ReviewCmd) submitToArgo(ctx *execcontext.Context, cloneBranch string) error {
	logger.Verbose("Submitting review Argo Workflow...")

	if err := git.IsBranchSyncedWithRemote(cloneBranch); err != nil {
		return err
	}

	wf, err := workflow.GenerateReviewWorkflow(ctx, cloneBranch)
	if err != nil {
		return fmt.Errorf("failed to generate review workflow: %w", err)
	}

	if ctx.IsVerbose() {
		workflowYAML, _ := wf.Render()
		logger.Verbosef("Generated workflow YAML:\n%s", workflowYAML)
	}

	workflowName, err := wf.Submit()
	if err != nil {
		return fmt.Errorf("failed to submit review workflow: %w", err)
	}

	logger.Successf("Review workflow submitted: %s", workflowName)
	logger.Infof("To follow logs, run: argo logs -n %s %s -f", wf.Namespace, workflowName)
	return nil
}

func (r *ReviewCmd) printStats() error {
	cmd := exec.Command("opencode", "stats")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *ReviewCmd) commitProjectFile(ctx *execcontext.Context, absProjectFile, reviewName, componentName string, itemIndex int, summaryPath string) error {
	commitMsg := r.buildCommitMessage(componentName, itemIndex, summaryPath)
	if err := run.CommitFileChanges(ctx, reviewName, absProjectFile, commitMsg); err != nil {
		return fmt.Errorf("failed to commit review findings: %w", err)
	}
	return nil
}

func (r *ReviewCmd) buildCommitMessage(componentName string, itemIndex int, summaryPath string) string {
	prefix := fmt.Sprintf("%s-%d", componentName, itemIndex)

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		logger.Verbosef("Failed to read summary file: %v", err)
		return fmt.Sprintf("review: %s", prefix)
	}

	summary := strings.TrimSpace(string(data))
	if summary == "" {
		return fmt.Sprintf("review: %s", prefix)
	}

	return fmt.Sprintf("review: %s %s", prefix, summary)
}

func (r *ReviewCmd) submitPR(ctx *execcontext.Context, absProjectFile, reviewName, baseBranch string) error {
	// Load project for PR title (fallback to reviewName)
	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		// If project file cannot be loaded, create a minimal project for PR creation
		proj = &project.Project{Name: reviewName}
	}

	body := fmt.Sprintf("AI code review findings for `%s`.", reviewName)
	generatedBody, err := run.GenerateReviewPRBody(ctx, absProjectFile)
	if err != nil {
		logger.Verbosef("Failed to generate PR body with AI: %v", err)
	} else {
		body = generatedBody
	}

	prURL, err := run.CreatePullRequest(ctx, proj, reviewName, baseBranch, body)
	if err != nil {
		if errors.Is(err, github.ErrNoCommitsBetweenBranches) {
			logger.Verbose("No commits ahead of base branch — skipping PR creation")
			return nil
		}
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	logger.Successf("Pull request created: %s", prURL)
	return nil
}

func (r *ReviewCmd) resolveModel(ralphConfig *config.RalphConfig) string {
	if r.Model != "" {
		return r.Model
	}
	if ralphConfig.Review.Model != "" {
		return ralphConfig.Review.Model
	}
	return ralphConfig.Model
}

func (r *ReviewCmd) loadItemContent(item config.ReviewItem) (string, error) {
	switch {
	case item.Text != "":
		return item.Text, nil
	case item.File != "":
		absPath, err := filepath.Abs(item.File)
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), nil
	case item.URL != "":
		resp, err := http.Get(item.URL)
		if err != nil {
			return "", fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("review item has no content")
	}
}

func (r *ReviewCmd) itemLabel(item config.ReviewItem) string {
	switch {
	case item.Text != "":
		firstLine := strings.SplitN(item.Text, "\n", 2)[0]
		if len(firstLine) > 80 {
			firstLine = firstLine[:77] + "..."
		}
		return firstLine
	case item.File != "":
		return filepath.Base(item.File)
	case item.URL != "":
		u, err := url.Parse(item.URL)
		if err == nil && u.Path != "" {
			return path.Base(u.Path)
		}
		return path.Base(item.URL)
	default:
		return ""
	}
}

type reviewPromptData struct {
	ConfigContent   string
	Project         string
	RalphProjectDoc string
	ReviewName      string
}

var reviewPromptTemplate = template.Must(template.New("review").Parse(reviewInstructions))

func (r *ReviewCmd) buildPrompt(content, projectPath, projectDoc, reviewName string) string {
	var buf bytes.Buffer
	data := reviewPromptData{
		ConfigContent:   content,
		Project:         projectPath,
		RalphProjectDoc: projectDoc,
		ReviewName:      reviewName,
	}
	reviewPromptTemplate.Execute(&buf, data)
	return buf.String()
}

func fetchRalphProjectDoc() (string, error) {
	resp, err := http.Get(ralphProjectDocURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (r *ReviewCmd) runOverview(ctx *execcontext.Context, overviewPath, projectPath, reviewName string) (*Overview, error) {
	prompt := buildOverviewPrompt(overviewPath)

	if r.Verbose {
		logger.Verbose(prompt)
	}

	logger.Verbose("Running overview step: generating code overview...")
	if err := ai.RunAgent(ctx, prompt); err != nil {
		return nil, fmt.Errorf("overview step failed: %w", err)
	}

	overview, err := loadOverview(overviewPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load overview: %w", err)
	}
	r.printDetectedComponents(overview)
	os.Remove(overviewPath)

	return overview, nil
}

func (r *ReviewCmd) runReview(ctx *execcontext.Context, overview *Overview, projectPath, projectDoc, reviewName, baseBranch string, ralphConfig *config.RalphConfig) (bool, error) {
	projectChanged := false

	seed := r.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
		logger.Infof("Using random seed: %d", seed)
	}

	// Shuffle components
	overview.Components = shuffleComponents(overview.Components, seed)

	// Shuffle review items with original indices
	itemsWithIdx := shuffleItemsWithIndices(ralphConfig.Review.Items, seed)

	for _, component := range overview.Components {
		for _, pair := range itemsWithIdx {
			item := pair.item
			i := pair.idx
			iterPrefix := fmt.Sprintf("%s-%d", component.Name, i)

			alreadyDone, err := git.BranchLogContainsPrefix(baseBranch, reviewName, iterPrefix)
			if err != nil {
				logger.Verbosef("Could not check commit log for prefix %s (continuing): %v", iterPrefix, err)
			}
			if alreadyDone {
				logger.Verbosef("Skipping component %s, item %d — prefix %s already in commit history", component.Name, i, iterPrefix)
				continue
			}

			label := r.itemLabel(item)
			logger.Infof("Component: %s, item: %s", component.Name, label)

			content, err := r.loadItemContent(item)
			if err != nil {
				return projectChanged, fmt.Errorf("failed to load review item %d: %w", i, err)
			}

			summaryPath, err := git.TmpPath(fmt.Sprintf("summary-%s-%d.txt", component.Name, i))
			if err != nil {
				return projectChanged, fmt.Errorf("failed to resolve summary path: %w", err)
			}

			prompt := buildComponentPrompt(content, projectPath, projectDoc, reviewName, component, summaryPath)

			if r.Verbose {
				logger.Verbose(prompt)
			}

			logger.Verbosef("Running component %s, review item %d/%d...", component.Name, i+1, len(itemsWithIdx))
			if err := ai.RunAgent(ctx, prompt); err != nil {
				return projectChanged, fmt.Errorf("component %s, item %d failed: %w", component.Name, i, err)
			}

			if git.IsFileModifiedOrNew(projectPath) {
				if err := r.commitProjectFile(ctx, projectPath, reviewName, component.Name, i, summaryPath); err != nil {
					return projectChanged, fmt.Errorf("failed to commit after component %s, item %d: %w", component.Name, i+1, err)
				}
				projectChanged = true
				os.Remove(summaryPath)
				// Submit PR immediately after first finding
				if err := r.submitPR(ctx, projectPath, reviewName, baseBranch); err != nil {
					logger.Warningf("Failed to create pull request: %v", err)
				}
				r.prSubmitted = true
				return projectChanged, nil
			}

			os.Remove(summaryPath)
		}
	}

	return projectChanged, nil
}

func (r *ReviewCmd) printDetectedComponents(overview *Overview) {
	for _, comp := range overview.Components {
		logger.Infof("Component: %s (%s) - %s", comp.Name, comp.Path, comp.Summary)
	}
}
