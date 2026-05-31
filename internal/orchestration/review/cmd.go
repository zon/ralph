package review

import (
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

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/review"
)

const ralphProjectDocURL = "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/skills/ralph-write-project.md"

type AIClient interface {
	BuildReviewItemPrompt(content string) (string, error)
	BuildLoopItemPrompt(content, funcName, funcPath string) (string, error)
	RunAgent(prompt string) error
	DisplayStats() error
	GenerateReviewPRBody(slug, title string, requirementSummaries []string) (string, error)
	SetModel(model string)
}

type GitClient interface {
	CurrentBranch() (string, error)
	HasUncommittedChanges() bool
	CommitAllAndPush(branch, commitMsg string) error
	DetectModifiedProjectFile(dir string) (string, error)
	IsBranchSyncedWithRemote(branch string) error
	TmpPath(filename string) (string, error)
}

type GitHubClient interface {
	CreatePullRequest(proj *project.Project, reviewName, baseBranch, body string) (string, error)
}

type WorkflowClient interface {
	SubmitReview(cloneBranch string) (string, error)
	FollowLogs(workflowName string) error
}

type ReviewFlags struct {
	Seed    int64
	Filter  string
	One     bool
	Base    string
	Local   bool
	Follow  bool
	Model   string
	Verbose bool
}

func (f ReviewFlags) Validate() error {
	if f.Follow && f.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}
	return nil
}

type ReviewCmd struct {
	ai       AIClient
	git      GitClient
	github   GitHubClient
	workflow WorkflowClient
}

func NewReviewCmd(ai AIClient, git GitClient, github GitHubClient, workflow WorkflowClient) *ReviewCmd {
	return &ReviewCmd{
		ai:       ai,
		git:      git,
		github:   github,
		workflow: workflow,
	}
}

func (r *ReviewCmd) Run(flags ReviewFlags) error {
	if flags.Verbose {
		logger.SetVerbose(true)
	}

	if err := flags.Validate(); err != nil {
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

	r.ai.SetModel(r.resolveModel(flags, ralphConfig))

	startingBranch, err := r.git.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if !flags.Local {
		return r.submitToArgo(flags, startingBranch)
	}

	branchName, detectedProjectFile, err := r.runReview(flags, ralphConfig)
	if err != nil {
		return fmt.Errorf("review step failed: %w", err)
	}

	if branchName != "" && branchName != startingBranch {
		absProjectFile, err := filepath.Abs(detectedProjectFile)
		if err != nil {
			return fmt.Errorf("failed to resolve project file path: %w", err)
		}

		baseBranch := resolveBaseBranch(flags.Base, startingBranch, branchName, ralphConfig.DefaultBranch)

		if err := r.submitPR(projInfo{absPath: absProjectFile, reviewName: branchName, baseBranch: baseBranch}); err != nil {
			logger.Warningf("Failed to create pull request: %v", err)
		}
	}

	if err := r.ai.DisplayStats(); err != nil {
		logger.Verbosef("Failed to print stats: %v", err)
	}

	return nil
}

type projInfo struct {
	absPath    string
	reviewName string
	baseBranch string
}

func (r *ReviewCmd) runReview(flags ReviewFlags, ralphConfig *config.RalphConfig) (branchName, detectedProjectFile string, err error) {
	seed := flags.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
		logger.Infof("Using random seed: %d", seed)
	}

	items := filterItems(ralphConfig.Review.Items, flags.Filter)
	if flags.Filter != "" && len(items) == 0 {
		return "", "", fmt.Errorf("no review items match filter %q", flags.Filter)
	}

	itemsWithIdx := shuffleItemsWithIndices(items, seed)
	if flags.One {
		itemsWithIdx = itemsWithIdx[:1]
	}

	for _, pair := range itemsWithIdx {
		item := pair.item
		i := pair.idx

		label := itemLabel(item)
		logger.Infof("Item: %s", label)

		if item.Loop != "" {
			branchName, detectedProjectFile, err = r.runLoopItem(flags, item, i, branchName, detectedProjectFile)
			if err != nil {
				return branchName, detectedProjectFile, fmt.Errorf("failed to run loop item %d: %w", i, err)
			}
			continue
		}

		content, err := r.loadItemContent(item)
		if err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("failed to load review item %d: %w", i, err)
		}

		prompt, err := r.ai.BuildReviewItemPrompt(content)
		if err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("failed to build review prompt: %w", err)
		}

		if flags.Verbose {
			logger.Verbose(prompt)
		}

		logger.Verbosef("Running review item %d/%d...", i+1, len(itemsWithIdx))
		if err := r.ai.RunAgent(prompt); err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("review item %d failed: %w", i, err)
		}

		currentProjectFile, err := r.git.DetectModifiedProjectFile("projects")
		if err != nil {
			logger.Verbosef("Failed to detect project file: %v", err)
		}

		if currentProjectFile != "" && branchName == "" {
			branchName = strings.TrimSuffix(filepath.Base(currentProjectFile), filepath.Ext(currentProjectFile))
			detectedProjectFile = currentProjectFile
		}

		if branchName != "" && r.git.HasUncommittedChanges() {
			if err := r.commitReviewItemChanges(branchName, i); err != nil {
				return branchName, detectedProjectFile, fmt.Errorf("failed to commit after item %d: %w", i+1, err)
			}
		}
	}

	return branchName, detectedProjectFile, nil
}

func (r *ReviewCmd) commitReviewItemChanges(branchName string, itemIndex int) error {
	summaryPath, err := r.git.TmpPath(fmt.Sprintf("summary-%d.txt", itemIndex))
	if err != nil {
		return fmt.Errorf("failed to resolve summary path: %w", err)
	}

	commitMsg := r.buildCommitMessage(itemIndex, summaryPath)

	if err := r.git.CommitAllAndPush(branchName, commitMsg); err != nil {
		return err
	}

	os.Remove(summaryPath)
	return nil
}

func (r *ReviewCmd) runLoopItem(flags ReviewFlags, item config.ReviewItem, itemIndex int, branchName, detectedProjectFile string) (string, string, error) {
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

		prompt, err := r.ai.BuildLoopItemPrompt(content, iteration.FunctionName, iteration.FunctionPath)
		if err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("failed to build loop prompt: %w", err)
		}

		if flags.Verbose {
			logger.Verbose(prompt)
		}

		logger.Verbosef("Running loop item %d iteration %d/%d...", itemIndex+1, iterationIdx+1, len(iterations))
		if err := r.ai.RunAgent(prompt); err != nil {
			return branchName, detectedProjectFile, fmt.Errorf("loop item %d iteration %d failed: %w", itemIndex, iterationIdx+1, err)
		}

		currentProjectFile, err := r.git.DetectModifiedProjectFile("projects")
		if err != nil {
			logger.Verbosef("Failed to detect project file: %v", err)
		}

		if currentProjectFile != "" && branchName == "" {
			branchName = strings.TrimSuffix(filepath.Base(currentProjectFile), filepath.Ext(currentProjectFile))
			detectedProjectFile = currentProjectFile
		}

		if branchName != "" && r.git.HasUncommittedChanges() {
			combinedIndex := itemIndex*100 + iterationIdx
			if err := r.commitReviewItemChanges(branchName, combinedIndex); err != nil {
				return branchName, detectedProjectFile, fmt.Errorf("failed to commit after loop item %d iteration %d: %w", itemIndex+1, iterationIdx+1, err)
			}
		}
	}

	return branchName, detectedProjectFile, nil
}

func (r *ReviewCmd) submitToArgo(flags ReviewFlags, cloneBranch string) error {
	logger.Verbose("Submitting review Argo Workflow...")

	if err := r.git.IsBranchSyncedWithRemote(cloneBranch); err != nil {
		return err
	}

	workflowName, err := r.workflow.SubmitReview(cloneBranch)
	if err != nil {
		return fmt.Errorf("failed to submit review workflow: %w", err)
	}

	logger.Successf("Review workflow submitted: %s", workflowName)

	if flags.Follow {
		if err := r.workflow.FollowLogs(workflowName); err != nil {
			return fmt.Errorf("argo logs failed: %w", err)
		}
	} else {
		logger.Infof("To follow logs, run: argo logs %s -f", workflowName)
	}
	return nil
}

func (r *ReviewCmd) submitPR(info projInfo) error {
	proj, err := project.LoadProject(info.absPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	var requirementSummaries []string
	for _, req := range proj.Requirements {
		reqSummary := req.Description
		if reqSummary == "" {
			reqSummary = req.Slug
		}
		if reqSummary != "" {
			requirementSummaries = append(requirementSummaries, reqSummary)
		}
	}

	body := fmt.Sprintf("AI code review findings for `%s`.", info.reviewName)

	generatedBody, err := r.ai.GenerateReviewPRBody(proj.Slug, proj.Title, requirementSummaries)
	if err != nil {
		logger.Verbosef("Failed to generate PR body with AI: %v", err)
	} else {
		body = generatedBody
	}

	prURL, err := r.github.CreatePullRequest(proj, info.reviewName, info.baseBranch, body)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	logger.Successf("Pull request created: %s", prURL)
	return nil
}

func (r *ReviewCmd) resolveModel(flags ReviewFlags, ralphConfig *config.RalphConfig) string {
	if flags.Model != "" {
		return flags.Model
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

func (r *ReviewCmd) buildCommitMessage(itemIndex int, summaryPath string) string {
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

func resolveBaseBranch(baseFlag, currentBranch, projectBranch, defaultBranch string) string {
	if baseFlag != "" {
		return baseFlag
	}
	if currentBranch != projectBranch {
		return currentBranch
	}
	return defaultBranch
}

func itemLabel(item config.ReviewItem) string {
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
