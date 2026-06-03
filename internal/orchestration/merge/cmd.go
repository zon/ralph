package merge

import (
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	ralphproj "github.com/zon/ralph/internal/project"
)

type WorkspaceSetupClient interface {
	Setup(flags wksp.WorkspaceFlags) error
}

type GitClient interface {
	CommitAndPush(message string) error
}

type GitHubClient interface {
	WaitForHeadSync(prBranch string) error
	MergePR(prNumber int) error
}

type ProjectClient interface {
	LoadAll() ([]*ralphproj.Project, error)
	FilterPassing(projects []*ralphproj.Project) []*ralphproj.Project
	DeleteAll(projects []*ralphproj.Project) error
}

func NewWorkflowMergeCmd(workspace WorkspaceSetupClient, git GitClient, github GitHubClient, project ProjectClient) *WorkflowMergeCmd {
	return &WorkflowMergeCmd{
		workspace: workspace,
		git:       git,
		github:    github,
		project:   project,
	}
}

type WorkflowMergeCmd struct {
	workspace WorkspaceSetupClient
	git       GitClient
	github    GitHubClient
	project   ProjectClient
}

type WorkflowMergeFlags struct {
	Repo        string
	CloneBranch string
	PRBranch    string
	PRNumber    int
	BotName     string
	BotEmail    string
}

func (f WorkflowMergeFlags) WorkspaceFlags() wksp.WorkspaceFlags {
	return wksp.WorkspaceFlags{
		Repo:        f.Repo,
		CloneBranch: f.CloneBranch,
		BotName:     f.BotName,
		BotEmail:    f.BotEmail,
	}
}

func (w *WorkflowMergeCmd) Merge(flags WorkflowMergeFlags) error {
	if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
		return err
	}
	pushed, err := w.cleanupCompletedProjects()
	if err != nil {
		return err
	}
	if pushed {
		if err := w.github.WaitForHeadSync(flags.PRBranch); err != nil {
			return err
		}
	}
	return w.github.MergePR(flags.PRNumber)
}

func (w *WorkflowMergeCmd) cleanupCompletedProjects() (bool, error) {
	projects, err := w.project.LoadAll()
	if err != nil {
		return false, err
	}
	completed := w.project.FilterPassing(projects)
	if len(completed) == 0 {
		return false, nil
	}
	if err := w.project.DeleteAll(completed); err != nil {
		return false, err
	}
	if err := w.git.CommitAndPush("chore: remove completed project files"); err != nil {
		return false, err
	}
	return true, nil
}
