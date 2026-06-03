package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/opencode"
	orchestrationComment "github.com/zon/ralph/internal/orchestration/comment"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/services"
)

type WorkflowCommentCmd struct {
	Repo             string `help:"GitHub repository (owner/repo)" required:""`
	CloneBranch      string `help:"Branch to clone"`
	ProjectBranch    string `help:"Project branch to checkout" name:"project-branch" required:""`
	BotName          string `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail         string `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	CommentBody      string `help:"Comment body text" required:""`
	PRNumber         int    `help:"Pull request number" name:"pr" required:""`
	RepoOwner        string `help:"Repository owner" name:"repo-owner" required:""`
	RepoName         string `help:"Repository name" name:"repo-name" required:""`
	NoServices       bool   `help:"Skip service startup" default:"false"`
	InstructionsFile string `help:"Path to instructions file" name:"instructions-file"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowCommentCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	ctx.SetNoServices(w.NoServices)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	cloneBranch := w.CloneBranch
	if cloneBranch == "" {
		cloneBranch = os.Getenv("GIT_BRANCH")
	}

	cmd := newOrchestrationWorkflowCommentCmd(ctx)
	flags := orchestrationComment.WorkflowCommentFlags{
		Repo:             w.Repo,
		CloneBranch:      cloneBranch,
		ProjectBranch:    w.ProjectBranch,
		BotName:          w.BotName,
		BotEmail:         w.BotEmail,
		CommentBody:      w.CommentBody,
		PRNumber:         w.PRNumber,
		RepoOwner:        w.RepoOwner,
		RepoName:         w.RepoName,
		NoServices:       w.NoServices,
		InstructionsFile: w.InstructionsFile,
	}
	return cmd.Run(flags)
}

func newOrchestrationWorkflowCommentCmd(ctx *execcontext.Context) *orchestrationComment.WorkflowCommentCmd {
	return orchestrationComment.NewWorkflowCommentCmd(
		&workspaceSetupAdapter{ctx: ctx},
		&configOptionalAdapter{},
		&workflowCommentAIClient{ctx: ctx},
		&workflowCommentServicesClient{ctx: ctx},
		&workflowCommentGitClient{},
		&workflowCommentGitHubClient{},
	)
}

// ---------------------------------------------------------------------------
// workflowCommentAIClient implements orchestration/comment.AIClient
// ---------------------------------------------------------------------------

type workflowCommentAIClient struct {
	ctx *execcontext.Context
}

func (c *workflowCommentAIClient) RenderCommentPrompt(ctx orchestrationComment.CommentContext, instructionsFile string) (string, error) {
	var instructions string
	if instructionsFile != "" {
		data, err := os.ReadFile(instructionsFile)
		if err != nil {
			return "", fmt.Errorf("failed to read instructions file: %w", err)
		}
		instructions = string(data)
	} else {
		instructions = config.DefaultCommentInstructions()
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Comment Context\n\n"))
	b.WriteString(fmt.Sprintf("Comment: %s\n", ctx.CommentBody))
	b.WriteString(fmt.Sprintf("PR Number: %d\n", ctx.PRNumber))
	b.WriteString(fmt.Sprintf("PR Branch: %s\n", ctx.PRBranch))
	b.WriteString(fmt.Sprintf("Repo: %s/%s\n", ctx.RepoOwner, ctx.RepoName))
	b.WriteString("\n")
	b.WriteString(instructions)

	return b.String(), nil
}

func (c *workflowCommentAIClient) RunAgent(prompt string) error {
	return ai.RunAgent(c.ctx, opencode.New(), prompt)
}

func (c *workflowCommentAIClient) GenerateChangelog() error {
	return ai.GenerateChangelog(c.ctx, opencode.New())
}

func (c *workflowCommentAIClient) GenerateCommentReply(ctx orchestrationComment.CommentContext, pushed bool) (string, error) {
	status := "completed successfully"
	if !pushed {
		status = "completed with no changes to push"
	}
	return fmt.Sprintf("AI agent %s for PR #%d", status, ctx.PRNumber), nil
}

// ---------------------------------------------------------------------------
// workflowCommentServicesClient implements orchestration/comment.ServicesClient
// ---------------------------------------------------------------------------

type workflowCommentServicesClient struct {
	ctx *execcontext.Context
}

func (c *workflowCommentServicesClient) Start(cfg *config.RalphConfig) (*services.Manager, error) {
	mgr := services.NewManager(c.ctx.Output())
	_, err := mgr.Start(cfg.Services)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func (c *workflowCommentServicesClient) Stop(svc *services.Manager) {
	if svc != nil {
		svc.Stop()
	}
}

// ---------------------------------------------------------------------------
// workflowCommentGitClient implements orchestration/comment.GitClient
// ---------------------------------------------------------------------------

type workflowCommentGitClient struct{}

func (c *workflowCommentGitClient) HasChanges() bool {
	return git.HasUncommittedChanges()
}

func (c *workflowCommentGitClient) ReportExists() bool {
	_, err := os.Stat("report.md")
	return err == nil
}

func (c *workflowCommentGitClient) CommitAndPushFromReport() error {
	return git.CommitChanges(true, "", "", "")
}

// ---------------------------------------------------------------------------
// workflowCommentGitHubClient implements orchestration/comment.GitHubClient
// ---------------------------------------------------------------------------

type workflowCommentGitHubClient struct{}

func (c *workflowCommentGitHubClient) PostComment(prNumber int, body string) error {
	cmd := exec.Command("gh", "pr", "comment", fmt.Sprintf("%d", prNumber), "--body", body)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to post comment on PR #%d: %w (output: %s)", prNumber, err, output)
	}
	return nil
}
