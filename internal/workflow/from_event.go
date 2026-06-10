package workflow

import (
	"path/filepath"
	"strings"

	githubpkg "github.com/zon/ralph/internal/github"
)

// WebhookEvent contains the fields from a filtered GitHub webhook event needed
// to construct a workflow.
type WebhookEvent struct {
	Body      string
	Approved  bool
	PRBranch  string
	RepoOwner string
	RepoName  string
	PRNumber  string
}

// WorkflowResult holds the output of FromWebhookEvent. Exactly one of Run or Merge is non-nil.
type WorkflowResult struct {
	Run       *Workflow
	Merge     *MergeWorkflow
	Namespace string
}

// FromWebhookEvent converts a webhook event into an Argo Workflow.
// Comment events produce a Run workflow that calls `ralph comment`.
// Approval events produce a MergeWorkflow that calls `ralph merge --local`.
func FromWebhookEvent(event WebhookEvent, opts WorkflowOptions) (*WorkflowResult, error) {
	projectFile := ProjectFileFromBranch(event.PRBranch)
	repoURL := githubpkg.CloneURL(event.RepoOwner, event.RepoName)

	if event.Approved {
		mw, err := GenerateMergeWorkflowWithGitInfo(repoURL, event.PRBranch, event.PRBranch, event.PRNumber, opts)
		if err != nil {
			return nil, err
		}
		return &WorkflowResult{Merge: mw, Namespace: opts.Namespace}, nil
	}

	projectName := strings.TrimSuffix(filepath.Base(projectFile), filepath.Ext(projectFile))
	repo, err := githubpkg.ParseRemoteURL(repoURL)
	if err != nil {
		return nil, err
	}
	wf := &Workflow{
		ProjectName:   projectName,
		Repo:          repo,
		CloneBranch:   event.PRBranch,
		ProjectBranch: event.PRBranch,
		ProjectPath:   projectFile,
		CommentBody:   event.Body,
		PRNumber:      event.PRNumber,
		Image:       opts.Image,
		KubeContext: opts.KubeContext,
		Namespace:   opts.Namespace,
	}
	return &WorkflowResult{Run: wf, Namespace: opts.Namespace}, nil
}

// ProjectFileFromBranch derives the project file path from the PR head branch name.
//
// Convention: branch "ralph/<project-name>" → "projects/<project-name>.yaml"
//
// If the branch does not follow the ralph/ prefix convention the full branch
// name (with slashes replaced by dashes) is used as the project name.
func ProjectFileFromBranch(branch string) string {
	projectName := branch
	if strings.HasPrefix(branch, "ralph/") {
		projectName = strings.TrimPrefix(branch, "ralph/")
	} else {
		projectName = strings.ReplaceAll(branch, "/", "-")
	}
	return filepath.Join("projects", projectName+".yaml")
}
