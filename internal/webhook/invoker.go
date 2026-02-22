package webhook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
	dryRun     bool
	LastInvoke *InvokeResult
}

// NewInvoker creates an Invoker. When dryRun is true the invoker records what
// it would have done instead of running real commands.
func NewInvoker(dryRun bool) *Invoker {
	return &Invoker{dryRun: dryRun}
}

// HandleEvent returns an EventHandler that invokes the appropriate ralph
// command in response to a webhook event.
func (inv *Invoker) HandleEvent() EventHandler {
	return func(eventType string, repoOwner, repoName string, payload map[string]interface{}) {
		prBranch, _ := nestedString(payload, "pull_request", "head", "ref")
		projectFile := projectFileFromBranch(prBranch)

		switch eventType {
		case "pull_request_review_comment":
			commentBody, _ := nestedString(payload, "comment", "body")
			_ = inv.InvokeRalphRun(projectFile, commentBody)

		case "pull_request_review":
			_ = inv.InvokeRalphMerge(projectFile, prBranch)
		}
	}
}

// InvokeRalphRun constructs a webhook-specific instructions file and invokes
// `ralph run <projectFile> --instructions <file> --remote --no-notify`.
// The instructions file is written to a temporary directory; it is the
// caller's responsibility to clean it up (the process is detached).
func (inv *Invoker) InvokeRalphRun(projectFile, commentBody string) error {
	instructions := buildInstructions(commentBody)

	if inv.dryRun {
		inv.LastInvoke = &InvokeResult{
			Command:             "run",
			Args:                []string{projectFile, "--instructions", "<temp>", "--remote", "--no-notify"},
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

	args := []string{projectFile, "--instructions", instructionsFile, "--remote", "--no-notify"}
	cmd := exec.Command("ralph", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ralph run: %w", err)
	}

	// Detach: we do not wait for completion.
	go func() {
		_ = cmd.Wait()
		_ = os.RemoveAll(tmpDir)
	}()

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

	cmd := exec.Command("ralph", "merge", projectFile, prBranch)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ralph merge: %w", err)
	}

	// Detach: we do not wait for completion.
	go func() {
		_ = cmd.Wait()
	}()

	return nil
}

// buildInstructions constructs the content of the webhook-specific instructions
// file for a comment event.
func buildInstructions(commentBody string) string {
	return fmt.Sprintf(`# Webhook Instructions

You have been triggered by a GitHub pull request comment. The comment text is:

---
%s
---

Your task:
1. Read the comment carefully.
2. If the comment asks a question, answer it by posting a GitHub PR comment.
3. If the comment requests code changes, implement them, then commit and push the changes.
4. After completing your work, post a GitHub PR comment summarising what you did.

When posting PR comments use the gh CLI:
  gh pr comment <number> --body "<your summary>"
`, commentBody)
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
