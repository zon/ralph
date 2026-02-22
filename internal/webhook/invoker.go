package webhook

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
)

// runDetached starts cmd, waits for it to finish in a goroutine, and logs any
// error together with the combined stdout+stderr output.  The label is used in
// log messages to identify which command failed.
func runDetached(cmd *exec.Cmd, label string, cleanup func()) {
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		logger.Verbosef("failed to start %s: %v", label, err)
		if cleanup != nil {
			cleanup()
		}
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			logger.Verbosef("%s failed: %v\n%s", label, err, out.String())
		}
		if cleanup != nil {
			cleanup()
		}
	}()
}

// InvokeResult captures what an invoker would have done, used in dry-run mode.
type InvokeResult struct {
	// Command is the ralph subcommand that would have been run ("run" or "merge").
	Command string
	// Args are the arguments that would have been passed to ralph.
	Args []string
	// InstructionsContent is the content that would have been written to the
	// instructions file (only set for "run" invocations).
	InstructionsContent string
}

// Invoker executes ralph CLI commands in response to webhook events.
// In dry-run mode no external process is started; instead the call is
// recorded in LastInvoke so tests can assert on it.
type Invoker struct {
	dryRun              bool
	commentInstructions string // template text; {{.CommentBody}} is replaced with the comment
	LastInvoke          *InvokeResult
}

// NewInvoker creates an Invoker. When dryRun is true the invoker records what
// it would have done instead of running real commands.
// commentInstructions is the template used to build per-comment instruction files;
// pass an empty string to use the embedded default.
func NewInvoker(dryRun bool, commentInstructions string) *Invoker {
	if commentInstructions == "" {
		commentInstructions = config.DefaultCommentInstructions
	}
	return &Invoker{dryRun: dryRun, commentInstructions: commentInstructions}
}

// HandleEvent returns an EventHandler that invokes the appropriate ralph
// command in response to a webhook event.
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
				_ = inv.InvokeRalphMerge(projectFile, prBranch)
			} else {
				commentBody, _ := nestedString(payload, "review", "body")
				_ = inv.InvokeRalphRun(projectFile, repoOwner, repoName, prBranch, commentBody)
			}
		}
	}
}

// InvokeRalphRun constructs a webhook-specific instructions file and invokes
// `ralph run <projectFile> --repo <owner/repo> --branch <branch> --instructions <file> --no-notify`.
// The instructions file is written to a temporary directory; it is the
// caller's responsibility to clean it up (the process is detached).
func (inv *Invoker) InvokeRalphRun(projectFile, repoOwner, repoName, branch, commentBody string) error {
	instructions := inv.buildInstructions(commentBody)
	repo := repoOwner + "/" + repoName

	if inv.dryRun {
		inv.LastInvoke = &InvokeResult{
			Command:             "run",
			Args:                []string{projectFile, "--repo", repo, "--branch", branch, "--instructions", "<temp>", "--no-notify"},
			InstructionsContent: instructions,
		}
		return nil
	}

	// Write instructions to a temp file.
	tmpDir, err := os.MkdirTemp("", "ralph-webhook-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for instructions: %w", err)
	}

	instructionsFile := filepath.Join(tmpDir, "instructions.md")
	if err := os.WriteFile(instructionsFile, []byte(instructions), 0600); err != nil {
		return fmt.Errorf("failed to write instructions file: %w", err)
	}

	args := []string{projectFile, "--repo", repo, "--branch", branch, "--instructions", instructionsFile, "--no-notify"}
	logger.Verbosef("invoking: ralph run %s", strings.Join(args, " "))
	cmd := exec.Command("ralph", args...)
	runDetached(cmd, "ralph run "+projectFile, func() { _ = os.RemoveAll(tmpDir) })
	return nil
}

// InvokeRalphMerge invokes `ralph merge <projectFile> <prBranch>`.
func (inv *Invoker) InvokeRalphMerge(projectFile, prBranch string) error {
	if inv.dryRun {
		inv.LastInvoke = &InvokeResult{
			Command: "merge",
			Args:    []string{projectFile, prBranch},
		}
		return nil
	}

	logger.Verbosef("invoking: ralph merge %s %s", projectFile, prBranch)
	cmd := exec.Command("ralph", "merge", projectFile, prBranch)
	runDetached(cmd, "ralph merge "+projectFile, nil)
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
