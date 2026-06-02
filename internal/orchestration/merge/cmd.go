package merge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/output"
)

type GitClient interface {
	CurrentBranch() (string, error)
	RevParse(rev string) (string, error)
	Push(branch string) error
}

type GitHubClient interface {
	MergePR(pr, repo string) error
	GetPRHeadRefOid(pr string) (string, error)
}

type ProjectClient interface {
	FindCompleteProjects(dir string) ([]string, error)
	RemoveAndCommit(files []string) error
}

type WorkflowClient interface {
	SubmitMergeWorkflow(branch string) (string, error)
}

type MergeFlags struct {
	Branch  string
	PR      string
	Repo    string
	Local   bool
	Verbose bool
}

type MergeCmd struct {
	git      GitClient
	github   GitHubClient
	project  ProjectClient
	workflow WorkflowClient
	out      *output.Client
}

func NewMergeCmd(git GitClient, github GitHubClient, project ProjectClient, workflow WorkflowClient, out *output.Client) *MergeCmd {
	return &MergeCmd{
		git:      git,
		github:   github,
		project:  project,
		workflow: workflow,
		out:      out,
	}
}

func (m *MergeCmd) SetOutput(out *output.Client) {
	m.out = out
}

func (m *MergeCmd) Run(flags MergeFlags) error {
	if flags.Verbose {
		logger.SetVerbose(true)
	}

	if flags.Local {
		return m.runLocal(flags)
	}

	workflowName, err := m.workflow.SubmitMergeWorkflow(flags.Branch)
	if err != nil {
		return fmt.Errorf("failed to submit merge workflow: %w", err)
	}

	logger.Successf("Merge workflow submitted: %s", workflowName)
	return nil
}

func (m *MergeCmd) runLocal(flags MergeFlags) error {
	if err := m.scanAndCleanupProjects(flags); err != nil {
		return err
	}

	return m.github.MergePR(flags.PR, flags.Repo)
}

func (m *MergeCmd) scanAndCleanupProjects(flags MergeFlags) error {
	projectsDir := "projects"
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		logger.Verbose("Projects directory not found, skipping complete project cleanup")
		return nil
	}

	completeProjects, err := m.project.FindCompleteProjects(projectsDir)
	if err != nil {
		return fmt.Errorf("failed to scan for complete projects: %w", err)
	}

	if len(completeProjects) == 0 {
		logger.Verbose("No complete projects found")
		return nil
	}

	logger.Infof("Found %d complete project(s) to clean up", len(completeProjects))
	for _, file := range completeProjects {
		relPath, err := filepath.Rel(".", file)
		if err != nil {
			relPath = file
		}
		logger.Infof("  - %s", relPath)
	}

	if err := m.project.RemoveAndCommit(completeProjects); err != nil {
		return fmt.Errorf("failed to remove complete projects: %w", err)
	}

	branch, err := m.git.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if err := m.git.Push(branch); err != nil {
		return fmt.Errorf("failed to push after removing complete projects: %w", err)
	}

	if err := m.waitForGitHubHead(flags.PR); err != nil {
		return fmt.Errorf("failed waiting for GitHub to sync push: %w", err)
	}

	return nil
}

func (m *MergeCmd) waitForGitHubHead(pr string) error {
	localSHA, err := m.git.RevParse("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get local HEAD: %w", err)
	}

	const maxAttempts = 20
	const pollInterval = 3 * time.Second

	for i := range maxAttempts {
		headRefOid, err := m.github.GetPRHeadRefOid(pr)
		if err != nil {
			return fmt.Errorf("failed to query PR head: %w", err)
		}

		if strings.HasPrefix(headRefOid, localSHA) || strings.HasPrefix(localSHA, headRefOid) {
			logger.Verbosef("GitHub head SHA matches local HEAD (%s)", localSHA[:8])
			return nil
		}

		if i < maxAttempts-1 {
			logger.Verbosef("Waiting for GitHub to sync push (attempt %d/%d, local=%s, remote=%s)...",
				i+1, maxAttempts, localSHA[:8], headRefOid[:8])
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("timed out waiting for GitHub to sync push (local HEAD: %s)", localSHA[:8])
}
