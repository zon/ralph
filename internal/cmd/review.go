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

//go:embed synthesis-instructions.md
var synthesisInstructions string

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

	if err := os.MkdirAll("projects", 0755); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
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
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(model)
	ctx.SetLocal(r.Local)
	ctx.SetKubeContext(r.Context)

	startingBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if !r.Local {
		return r.submitToArgo(ctx, startingBranch)
	}

	reviewName := ""

	overview, err := r.runOverview(ctx, overviewPath, "", "")
	if err != nil {
		return fmt.Errorf("overview step failed: %w", err)
	}

	projectChanged, detectedProjectFile, err := r.runReview(ctx, overview, projectDoc, &reviewName, ralphConfig)
	if err != nil {
		return fmt.Errorf("review step failed: %w", err)
	}

	if projectChanged && !r.prSubmitted {
		absProjectFile, err := filepath.Abs(detectedProjectFile)
		if err != nil {
			return fmt.Errorf("failed to resolve project file path: %w", err)
		}

		if reviewName == "" {
			reviewName = strings.TrimSuffix(filepath.Base(detectedProjectFile), filepath.Ext(detectedProjectFile))
		}

		baseBranch := resolveBaseBranch(r.Base, startingBranch, reviewName, ralphConfig.DefaultBranch)
		ctx.SetBaseBranch(baseBranch)

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
	return ai.DisplayStats()
}

func (r *ReviewCmd) commitProjectFile(ctx *execcontext.Context, absProjectFile, reviewName, componentName string, itemIndex int, summaryPath string) error {
	commitMsg := r.buildCommitMessage(componentName, itemIndex, summaryPath)
	return run.CommitFileAndPush(ctx, absProjectFile, reviewName, commitMsg)
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
	title := reviewName
	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		// If project cannot be loaded, create a minimal project for PR creation
		proj = &project.Project{Name: reviewName, Description: title}
	}

	var requirementSummaries []string
	for _, req := range proj.Requirements {
		reqSummary := req.Description
		if reqSummary == "" {
			reqSummary = req.Name
		}
		if reqSummary != "" {
			requirementSummaries = append(requirementSummaries, reqSummary)
		}
	}

	body := fmt.Sprintf("AI code review findings for `%s`.", reviewName)

	generatedBody, err := run.GenerateReviewPRBody(ctx, proj, requirementSummaries)
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

type synthesisPromptData struct {
	SummaryPath string
}

var synthesisPromptTemplate = template.Must(template.New("synthesis").Parse(synthesisInstructions))

func buildSynthesisPrompt(summaryPath string) string {
	var buf bytes.Buffer
	data := synthesisPromptData{SummaryPath: summaryPath}
	synthesisPromptTemplate.Execute(&buf, data)
	return buf.String()
}

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

func (r *ReviewCmd) runOverview(ctx *execcontext.Context, overviewPath, projectDoc, reviewName string) (*Overview, error) {
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

func (r *ReviewCmd) runReview(ctx *execcontext.Context, overview *Overview, projectDoc string, reviewName *string, ralphConfig *config.RalphConfig) (bool, string, error) {
	projectChanged := false
	detectedProjectFile := ""
	var detectedProjectFiles []string

	seed := r.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
		logger.Infof("Using random seed: %d", seed)
	}

	allComponents := overview.AllComponents()
	shuffledComponents := shuffleComponents(allComponents, seed)
	itemsWithIdx := shuffleItemsWithIndices(ralphConfig.Review.Items, seed)

	for _, component := range shuffledComponents {
		for _, pair := range itemsWithIdx {
			item := pair.item
			i := pair.idx

			label := r.itemLabel(item)
			logger.Infof("Component: %s, item: %s", component.Name, label)

			content, err := r.loadItemContent(item)
			if err != nil {
				return projectChanged, detectedProjectFile, fmt.Errorf("failed to load review item %d: %w", i, err)
			}

			summaryPath, err := git.TmpPath(fmt.Sprintf("summary-%s-%d.txt", component.Name, i))
			if err != nil {
				return projectChanged, detectedProjectFile, fmt.Errorf("failed to resolve summary path: %w", err)
			}

			prompt := buildComponentPrompt(content, projectDoc, component, summaryPath)

			if r.Verbose {
				logger.Verbose(prompt)
			}

			logger.Verbosef("Running component %s, review item %d/%d...", component.Name, i+1, len(itemsWithIdx))
			if err := ai.RunAgent(ctx, prompt); err != nil {
				return projectChanged, detectedProjectFile, fmt.Errorf("component %s, item %d failed: %w", component.Name, i, err)
			}

			newProjectFiles, err := git.DetectAllModifiedProjectFiles("projects")
			if err != nil {
				logger.Verbosef("Failed to detect project files: %v", err)
			}

			for _, pf := range newProjectFiles {
				if !containsProjectFile(detectedProjectFiles, pf) {
					detectedProjectFiles = append(detectedProjectFiles, pf)
				}
			}

			if len(detectedProjectFiles) > 0 && !r.prSubmitted {
				firstProject := detectedProjectFiles[0]
				*reviewName = strings.TrimSuffix(filepath.Base(firstProject), filepath.Ext(firstProject))

				if err := r.commitProjectFile(ctx, firstProject, *reviewName, component.Name, i, summaryPath); err != nil {
					return projectChanged, detectedProjectFile, fmt.Errorf("failed to commit after component %s, item %d: %w", component.Name, i+1, err)
				}
				projectChanged = true
				os.Remove(summaryPath)
				r.prSubmitted = true
				return projectChanged, firstProject, nil
			}

			os.Remove(summaryPath)
		}
	}

	if len(detectedProjectFiles) > 1 && !r.prSubmitted {
		synthesizedProject, err := r.runSynthesis(ctx, detectedProjectFiles)
		if err != nil {
			logger.Verbosef("Synthesis failed: %v", err)
		}
		if synthesizedProject != "" {
			*reviewName = strings.TrimSuffix(filepath.Base(synthesizedProject), filepath.Ext(synthesizedProject))
			projectChanged = true
			detectedProjectFile = synthesizedProject
		}
	}

	return projectChanged, detectedProjectFile, nil
}

func containsProjectFile(files []string, target string) bool {
	for _, f := range files {
		if f == target {
			return true
		}
	}
	return false
}

func (r *ReviewCmd) runSynthesis(ctx *execcontext.Context, projectFiles []string) (string, error) {
	summaryPath, err := git.TmpPath("synthesis-summary.txt")
	if err != nil {
		return "", fmt.Errorf("failed to resolve synthesis summary path: %w", err)
	}

	prompt := buildSynthesisPrompt(summaryPath)

	if r.Verbose {
		logger.Verbose(prompt)
	}

	logger.Verbose("Running cross-component synthesis...")
	if err := ai.RunAgent(ctx, prompt); err != nil {
		return "", fmt.Errorf("synthesis failed: %w", err)
	}

	synthesizedFiles, err := git.DetectAllModifiedProjectFiles("projects")
	if err != nil {
		logger.Verbosef("Failed to detect project files after synthesis: %v", err)
	}

	var synthesizedProject string
	if len(synthesizedFiles) > 0 {
		synthesizedProject = synthesizedFiles[0]
	}

	os.Remove(summaryPath)
	return synthesizedProject, nil
}

func (r *ReviewCmd) printDetectedComponents(overview *Overview) {
	if len(overview.Modules) > 0 {
		logger.Infof("Modules (%d):", len(overview.Modules))
		for _, mod := range overview.Modules {
			logger.Infof("  - %s (%s) - %s", mod.Name, mod.Path, mod.Summary)
		}
	}
	if len(overview.Apps) > 0 {
		logger.Infof("Apps (%d):", len(overview.Apps))
		for _, app := range overview.Apps {
			logger.Infof("  - %s (%s) - %s", app.Name, app.Path, app.Summary)
		}
	}
}
