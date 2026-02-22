package webhook

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/workflow"
)

// InvokeResult captures what an invoker would have done, used in dry-run mode.
type InvokeResult struct {
	// Command is the ralph subcommand that would have been run ("run" or "merge").
	Command string
	// WorkflowYAML is the generated Argo Workflow YAML (only set in dry-run mode).
	WorkflowYAML string
	// InstructionsContent is the rendered instructions passed to the workflow
	// (only set for "run" invocations).
	InstructionsContent string
}

// Invoker submits Argo Workflows in response to webhook events.
// In dry-run mode no workflow is submitted; instead the call is
// recorded in LastInvoke so tests can assert on it.
type Invoker struct {
	dryRun              bool
	commentInstructions string // template text; {{.CommentBody}} is replaced with the comment
	LastInvoke          *InvokeResult
}

// NewInvoker creates an Invoker. When dryRun is true the invoker records what
// it would have done instead of submitting real workflows.
// commentInstructions is the template used to build per-comment instruction files;
// pass an empty string to use the embedded default.
func NewInvoker(dryRun bool, commentInstructions string) *Invoker {
	if commentInstructions == "" {
		commentInstructions = config.DefaultCommentInstructions
	}
	return &Invoker{dryRun: dryRun, commentInstructions: commentInstructions}
}

// HandleEvent returns an EventHandler that submits the appropriate Argo Workflow
// in response to a webhook event.
func (inv *Invoker) HandleEvent() EventHandler {
	return func(eventType string, repoOwner, repoName string, payload map[string]interface{}) {
		prBranch, _ := nestedString(payload, "pull_request", "head", "ref")
		projectFile := projectFileFromBranch(prBranch)

		switch eventType {
		case "issue_comment", "pull_request_review_comment":
			commentBody, _ := nestedString(payload, "comment", "body")
			_ = inv.InvokeRalphRun(projectFile, repoOwner, repoName, prBranch, commentBody)

		case "pull_request_review":
			state, _ := nestedString(payload, "review", "state")
			if strings.ToLower(state) == "approved" {
				_ = inv.InvokeRalphMerge(projectFile, repoOwner, repoName, prBranch)
			} else {
				commentBody, _ := nestedString(payload, "review", "body")
				_ = inv.InvokeRalphRun(projectFile, repoOwner, repoName, prBranch, commentBody)
			}
		}
	}
}

// InvokeRalphRun generates and submits an Argo Workflow for a run operation.
// The comment body is rendered through the comment instructions template and
// passed directly as the workflow's instructions-md parameter.
func (inv *Invoker) InvokeRalphRun(projectFile, repoOwner, repoName, branch, commentBody string) error {
	instructions := inv.buildInstructions(commentBody)
	repo := repoOwner + "/" + repoName
	repoURL := "https://github.com/" + repo + ".git"
	projectName := strings.TrimSuffix(filepath.Base(projectFile), filepath.Ext(projectFile))

	ctx := &execcontext.Context{
		ProjectFile:    projectFile,
		Repo:           repo,
		NoNotify:       true,
		InstructionsMD: instructions,
	}

	if inv.dryRun {
		wf, _ := workflow.GenerateWorkflowWithGitInfo(ctx, projectName, repoURL, branch, branch, projectFile, false, false)
		var workflowYAML string
		if wf != nil {
			workflowYAML, _ = wf.Render()
		}
		inv.LastInvoke = &InvokeResult{
			Command:             "run",
			WorkflowYAML:        workflowYAML,
			InstructionsContent: instructions,
		}
		return nil
	}

	go func() {
		wf, err := workflow.GenerateWorkflowWithGitInfo(ctx, projectName, repoURL, branch, branch, projectFile, false, false)
		if err != nil {
			logger.Verbosef("InvokeRalphRun: failed to generate workflow for %s: %v", projectFile, err)
			return
		}
		name, err := wf.Submit()
		if err != nil {
			logger.Verbosef("InvokeRalphRun: failed to submit workflow for %s: %v", projectFile, err)
			return
		}
		logger.Verbosef("InvokeRalphRun: submitted workflow %s for %s", name, projectFile)
	}()
	return nil
}

// InvokeRalphMerge generates and submits an Argo Workflow for a merge operation.
func (inv *Invoker) InvokeRalphMerge(projectFile, repoOwner, repoName, prBranch string) error {
	if inv.dryRun {
		inv.LastInvoke = &InvokeResult{
			Command: "merge",
		}
		return nil
	}

	repo := repoOwner + "/" + repoName
	repoURL := "https://github.com/" + repo + ".git"

	go func() {
		mw, err := workflow.GenerateMergeWorkflowWithGitInfo(repoURL, prBranch, prBranch, projectFile)
		if err != nil {
			logger.Verbosef("InvokeRalphMerge: failed to generate workflow for %s: %v", projectFile, err)
			return
		}
		name, err := mw.Submit()
		if err != nil {
			logger.Verbosef("InvokeRalphMerge: failed to submit workflow for %s: %v", projectFile, err)
			return
		}
		logger.Verbosef("InvokeRalphMerge: submitted workflow %s for %s", name, projectFile)
	}()
	return nil
}

// buildInstructions renders the comment instructions template with the given comment body.
func (inv *Invoker) buildInstructions(commentBody string) string {
	tmpl, err := template.New("comment").Parse(inv.commentInstructions)
	if err != nil {
		// Fallback: return the raw template with the comment appended.
		return inv.commentInstructions + "\n\n" + commentBody
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ CommentBody string }{commentBody}); err != nil {
		return inv.commentInstructions + "\n\n" + commentBody
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
