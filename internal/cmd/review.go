package cmd

import (
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
	"time"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/review"
	"github.com/zon/ralph/internal/workflow"
)

const ralphProjectDocURL = "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/projects.md"

type itemWithIndex struct {
	item config.ReviewItem
	idx  int
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

func filterItems(items []config.ReviewItem, filter string) []config.ReviewItem {
	if filter == "" {
		return items
	}
	filterLower := strings.ToLower(filter)
	var filtered []config.ReviewItem
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Text), filterLower) ||
			strings.Contains(strings.ToLower(item.File), filterLower) ||
			strings.Contains(strings.ToLower(item.URL), filterLower) ||
			strings.Contains(strings.ToLower(item.Loop), filterLower) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

type ReviewRunCmd struct {
	Model       string `help:"Override the AI model from config" name:"model" optional:""`
	Base        string `help:"Override the base branch for PR creation" name:"base" optional:"" short:"B"`
	Local       bool   `help:"Run on this machine instead of submitting to Argo Workflows" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`
	Context     string `help:"Kubernetes context to use" name:"context" optional:""`
	Seed        int64  `help:"Random seed for shuffling review items (0 = random)" default:"0"`
	Follow      bool   `help:"Follow workflow logs after submission (only applicable without --local)" short:"f" default:"false"`
	Filter      string `help:"Only run review items whose text, file, or url property contains this string" name:"filter" optional:""`
	One         bool   `help:"Randomly pick one review item and run only that one" name:"one" default:"false"`
	prSubmitted bool
}

type ReviewCmd struct {
	Run ReviewRunCmd `cmd:"" default:"withargs" help:"Run AI-powered code reviews from config prompts"`
}

type ReviewFlags struct {
	Follow bool
	Local  bool
}

func (f ReviewFlags) Validate() error {
	if f.Follow && f.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}
	return nil
}

func (r *ReviewRunCmd) validateFlagCombinations() error {
	return ReviewFlags{
		Follow: r.Follow,
		Local:  r.Local,
	}.Validate()
}

func (r *ReviewRunCmd) Run() error {
	if r.Verbose {
		logger.SetVerbose(true)
	}

	if err := r.validateFlagCombinations(); err != nil {
		return err
	}

	if err := os.MkdirAll("projects", 0755); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
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
	ctx.SetFollow(r.Follow)
	ctx.SetKubeContext(r.Context)
	ctx.SetFilter(r.Filter)

	startingBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if !r.Local {
		return r.submitToArgo(ctx, startingBranch)
	}

	branchName, detectedProjectFile, err := r.runReview(ctx, ralphConfig, projectDoc)
	if err != nil {
		return fmt.Errorf("review step failed: %w", err)
	}

	if branchName != "" && branchName != startingBranch {
		absProjectFile, err := filepath.Abs(detectedProjectFile)
		if err != nil {
			return fmt.Errorf("failed to resolve project file path: %w", err)
		}

		baseBranch := resolveBaseBranch(r.Base, startingBranch, branchName, ralphConfig.DefaultBranch)
		ctx.SetBaseBranch(baseBranch)

		if err := r.submitPR(ctx, absProjectFile, branchName, baseBranch); err != nil {
			logger.Warningf("Failed to create pull request: %v", err)
		} else {
			r.prSubmitted = true
		}
	}

	if err := r.printStats(); err != nil {
		logger.Verbosef("Failed to print stats: %v", err)
	}

	return nil
}

func (r *ReviewRunCmd) runReview(ctx *execcontext.Context, ralphConfig *config.RalphConfig, projectDoc string) (branchName, detectedProjectFile string, err error) {
	seed := r.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
		logger.Infof("Using random seed: %d", seed)
	}

	items := filterItems(ralphConfig.Review.Items, r.Filter)
	if r.Filter != "" && len(items) == 0 {
		return "", "", fmt.Errorf("no review items match filter %q", r.Filter)
	}

	itemsWithIdx := shuffleItemsWithIndices(items, seed)
	if r.One {
		itemsWithIdx = itemsWithIdx[:1]
	}

	for _, pair := range itemsWithIdx {
		item := pair.item
		i := pair.idx

		label := r.itemLabel(item)
		logger.Infof("Item: %s", label)

		if item.Loop != "" {
			branchName, detectedProjectFile, err = r.runLoopItem(ctx, item, i, branchName, detectedProjectFile)
			if err != nil {
				return branchName, detectedProjectFile, fmt.Errorf("failed to run loop item %d: %w", i, err)
			}
			continue
		}

		content, err := r.loadItemContent(item)
		if err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("failed to load review item %d: %w", i, err)
		}

		prompt, err := ai.BuildReviewItemPrompt(content)
		if err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("failed to build review prompt: %w", err)
		}

		if r.Verbose {
			logger.Verbose(prompt)
		}

		logger.Verbosef("Running review item %d/%d...", i+1, len(itemsWithIdx))
		if err := ai.RunAgent(ctx, prompt); err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("review item %d failed: %w", i, err)
		}

		currentProjectFile, err := git.DetectModifiedProjectFile("projects")
		if err != nil {
			logger.Verbosef("Failed to detect project file: %v", err)
		}

		if currentProjectFile != "" && branchName == "" {
			branchName = strings.TrimSuffix(filepath.Base(currentProjectFile), filepath.Ext(currentProjectFile))
			detectedProjectFile = currentProjectFile
		}

		if branchName != "" && git.HasUncommittedChanges() {
			if err := r.commitReviewItemChanges(ctx, branchName, i); err != nil {
				return branchName, detectedProjectFile, fmt.Errorf("failed to commit after item %d: %w", i+1, err)
			}
		}
	}

	return branchName, detectedProjectFile, nil
}

func (r *ReviewRunCmd) commitReviewItemChanges(ctx *execcontext.Context, branchName string, itemIndex int) error {
	summaryPath, err := git.TmpPath(fmt.Sprintf("summary-%d.txt", itemIndex))
	if err != nil {
		return fmt.Errorf("failed to resolve summary path: %w", err)
	}

	commitMsg := r.buildCommitMessage(itemIndex, summaryPath)

	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	if err := git.CommitAllAndPush(auth, branchName, commitMsg); err != nil {
		return err
	}

	os.Remove(summaryPath)
	return nil
}

func (r *ReviewRunCmd) runLoopItem(ctx *execcontext.Context, item config.ReviewItem, itemIndex int, branchName, detectedProjectFile string) (string, string, error) {
	content, err := r.loadItemContent(item)
	if err != nil {
		return branchName, detectedProjectFile, fmt.Errorf("failed to load loop item content: %w", err)
	}

	iterations, err := review.ExpandLoop(item.Loop, "architecture.yaml")
	if err != nil {
		return branchName, detectedProjectFile, fmt.Errorf("failed to expand loop: %w", err)
	}

	logger.Verbosef("Loop item has %d iterations", len(iterations))

	for iterationIdx, iteration := range iterations {
		logger.Infof("Loop iteration %d/%d: %s (%s)", iterationIdx+1, len(iterations), iteration.FunctionName, iteration.FunctionPath)

		prompt, err := ai.BuildLoopItemPrompt(content, iteration.FunctionName, iteration.FunctionPath)
		if err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("failed to build loop prompt: %w", err)
		}

		if r.Verbose {
			logger.Verbose(prompt)
		}

		logger.Verbosef("Running loop item %d iteration %d/%d...", itemIndex+1, iterationIdx+1, len(iterations))
		if err := ai.RunAgent(ctx, prompt); err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("loop item %d iteration %d failed: %w", itemIndex, iterationIdx+1, err)
		}

		currentProjectFile, err := git.DetectModifiedProjectFile("projects")
		if err != nil {
			logger.Verbosef("Failed to detect project file: %v", err)
		}

		if currentProjectFile != "" && branchName == "" {
			branchName = strings.TrimSuffix(filepath.Base(currentProjectFile), filepath.Ext(currentProjectFile))
			detectedProjectFile = currentProjectFile
		}

		if branchName != "" && git.HasUncommittedChanges() {
			combinedIndex := itemIndex*100 + iterationIdx
			if err := r.commitReviewItemChanges(ctx, branchName, combinedIndex); err != nil {
				return branchName, detectedProjectFile, fmt.Errorf("failed to commit after loop item %d iteration %d: %w", itemIndex+1, iterationIdx+1, err)
			}
		}
	}

	return branchName, detectedProjectFile, nil
}

func (r *ReviewRunCmd) submitToArgo(ctx *execcontext.Context, cloneBranch string) error {
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

	if ctx.ShouldFollow() {
		if err := argo.FollowLogs(wf.Namespace, workflowName, wf.KubeContext); err != nil {
			return fmt.Errorf("argo logs failed: %w", err)
		}
	} else {
		logger.Infof("To follow logs, run: argo logs -n %s %s -f", wf.Namespace, workflowName)
	}
	return nil
}

func (r *ReviewRunCmd) printStats() error {
	return ai.DisplayStats()
}

func (r *ReviewRunCmd) buildCommitMessage(itemIndex int, summaryPath string) string {
	prefix := fmt.Sprintf("item-%d", itemIndex)

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

func (r *ReviewRunCmd) submitPR(ctx *execcontext.Context, absProjectFile, reviewName, baseBranch string) error {
	title := reviewName
	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
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

	generatedBody, err := ai.GenerateReviewPRBody(ctx, proj.Name, proj.Description, requirementSummaries)
	if err != nil {
		logger.Verbosef("Failed to generate PR body with AI: %v", err)
	} else {
		body = generatedBody
	}

	prURL, err := github.CreatePullRequest(ctx, proj, reviewName, baseBranch, body)
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

func (r *ReviewRunCmd) resolveModel(ralphConfig *config.RalphConfig) string {
	if r.Model != "" {
		return r.Model
	}
	if ralphConfig.Review.Model != "" {
		return ralphConfig.Review.Model
	}
	return ralphConfig.Model
}

func (r *ReviewRunCmd) loadItemContent(item config.ReviewItem) (string, error) {
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

func (r *ReviewRunCmd) itemLabel(item config.ReviewItem) string {
	if item.Loop != "" {
		return fmt.Sprintf("loop:%s", item.Loop)
	}
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
