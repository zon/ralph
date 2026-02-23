package webhook

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/webhookconfig"
	"github.com/zon/ralph/internal/workflow"
)

// Event represents a filtered GitHub webhook event — either a comment or a review.
// Use IsComment and IsReview to distinguish them.
type Event struct {
	Body      string // Comment or review body text
	Approved  bool   // True only for approved pull_request_review events
	PRBranch  string // Head branch of the pull request
	RepoOwner string
	RepoName  string
	PRNumber  string
	Author    string // GitHub login of the commenter or reviewer
}

// IsComment reports whether the event is a comment (not an approval).
func (e Event) IsComment() bool {
	return !e.Approved
}

// IsReview reports whether the event is an approved pull request review.
func (e Event) IsReview() bool {
	return e.Approved
}

// WorkflowResult holds the output of ToWorkflow. Exactly one of Run or Merge is non-nil.
type WorkflowResult struct {
	Run       *workflow.Workflow
	Merge     *workflow.MergeWorkflow
	Namespace string // Kubernetes namespace for workflow submission, from RepoConfig
}

// ToWorkflow converts the event into an Argo Workflow.
// cfg supplies the comment instructions template for run (comment) events;
// it is unused for approval (merge) events.
func (e Event) ToWorkflow(cfg *webhookconfig.Config) (*WorkflowResult, error) {
	projectFile := projectFileFromBranch(e.PRBranch)
	repoURL := "https://github.com/" + e.RepoOwner + "/" + e.RepoName + ".git"

	namespace := ""
	if repo := cfg.RepoByFullName(e.RepoOwner, e.RepoName); repo != nil {
		namespace = repo.Namespace
	}

	if e.Approved {
		mw, err := workflow.GenerateMergeWorkflowWithGitInfo(repoURL, e.PRBranch, e.PRBranch, projectFile)
		if err != nil {
			return nil, err
		}
		mw.Instructions = renderInstructions(cfg.App.MergeInstructions, e)
		return &WorkflowResult{Merge: mw, Namespace: namespace}, nil
	}

	instructions := renderInstructions(cfg.App.CommentInstructions, e)
	projectName := strings.TrimSuffix(filepath.Base(projectFile), filepath.Ext(projectFile))
	ctx := &execcontext.Context{
		ProjectFile:    projectFile,
		Repo:           e.RepoOwner + "/" + e.RepoName,
		NoNotify:       true,
		InstructionsMD: instructions,
	}
	wf, err := workflow.GenerateWorkflowWithGitInfo(ctx, projectName, repoURL, e.PRBranch, e.PRBranch, projectFile, false, false)
	if err != nil {
		return nil, err
	}
	return &WorkflowResult{Run: wf, Namespace: namespace}, nil
}

// renderInstructions renders the comment instructions template with event context.
func renderInstructions(tmplText string, e Event) string {
	tmpl, err := template.New("comment").Parse(tmplText)
	if err != nil {
		return tmplText + "\n\n" + e.Body
	}
	data := struct {
		CommentBody string
		PRNumber    string
		PRBranch    string
		RepoOwner   string
		RepoName    string
	}{
		CommentBody: e.Body,
		PRNumber:    e.PRNumber,
		PRBranch:    e.PRBranch,
		RepoOwner:   e.RepoOwner,
		RepoName:    e.RepoName,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return tmplText + "\n\n" + e.Body
	}
	return buf.String()
}

// projectFileFromBranch derives the project file path from the PR head branch name.
//
// Convention: branch "ralph/<project-name>" → "projects/<project-name>.yaml"
//
// If the branch does not follow the ralph/ prefix convention the full branch
// name (with slashes replaced by dashes) is used as the project name.
func projectFileFromBranch(branch string) string {
	projectName := branch
	if strings.HasPrefix(branch, "ralph/") {
		projectName = strings.TrimPrefix(branch, "ralph/")
	} else {
		projectName = strings.ReplaceAll(branch, "/", "-")
	}
	return filepath.Join("projects", projectName+".yaml")
}

