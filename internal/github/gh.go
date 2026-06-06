package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/zon/ralph/internal/output"
)

// ErrNoCommitsBetweenBranches is returned when gh pr create fails because the
// head branch has no commits ahead of the base branch. This is not an error in
// the traditional sense — it means the work was already complete before this
// run started, so there is nothing to open a PR for.
var ErrNoCommitsBetweenBranches = errors.New("no commits between branches")

// GHClient is the interface for GitHub CLI operations.
type GHClient interface {
	IsReady() bool
	FindExistingPR(head string) (string, error)
	CreatePR(title, body, base, head string) (string, error)
	GetPRHeadRefOid(pr string) (string, error)
	MergePR(pr, repo string) error
	ListCollaborators(ctx context.Context, owner, repo string) ([]string, error)
	RegisterWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error
}

// GH implements GHClient by shelling out to the gh CLI.
type GH struct {
	out *output.Client
}

func NewGH(out *output.Client) *GH {
	return &GH{out: out}
}

func (g *GH) IsReady() bool {
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}

	cmd = exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func (g *GH) FindExistingPR(head string) (string, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--head", head,
		"--state", "open",
		"--json", "url",
		"--limit", "1",
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to check for existing PRs: %w", err)
	}

	output := out.String()
	if !strings.Contains(output, "url") {
		return "", nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http") {
			return trimmed, nil
		}
	}

	return "", nil
}

func (g *GH) CreatePR(title, body, base, head string) (string, error) {
	existingPR, err := g.FindExistingPR(head)
	if err != nil {
		return "", err
	}

	if existingPR != "" {
		return updateExistingPR(g.out, existingPR, title, body)
	}

	cmd := exec.Command("gh", "pr", "create",
		"--title", title,
		"--body", body,
		"--base", base,
		"--head", head,
	)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if createErr := cmd.Run(); createErr != nil {
		return handleExistingPR(g.out, createErr, errOut.String(), out.String(), title, body)
	}

	return parsePRURL(g.out, out.String())
}

func (g *GH) GetPRHeadRefOid(pr string) (string, error) {
	cmd := exec.Command("gh", "pr", "view", pr, "--json", "headRefOid")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to query PR head: %w (output: %s)", err, out.String())
	}

	var result struct {
		HeadRefOid string `json:"headRefOid"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return "", fmt.Errorf("failed to parse PR head response: %w", err)
	}
	return result.HeadRefOid, nil
}

func (g *GH) MergePR(pr, repo string) error {
	autoArgs := []string{"pr", "merge", pr, "--merge", "--delete-branch", "--auto"}
	if repo != "" {
		autoArgs = append(autoArgs, "--repo", repo)
	}
	var autoOut bytes.Buffer
	autoCmd := exec.Command("gh", autoArgs...)
	autoCmd.Stdout = os.Stdout
	autoCmd.Stderr = &autoOut
	if err := autoCmd.Run(); err != nil {
		autoErrStr := autoOut.String()
		if strings.Contains(autoErrStr, "clean status") || strings.Contains(autoErrStr, "Protected branch rules not configured") || strings.Contains(autoErrStr, "enablePullRequestAutoMerge") {
			g.out.Debugf("PR #%s is already mergeable, merging immediately", pr)
			return mergePRImmediate(g.out, pr, repo)
		}
		fmt.Fprint(os.Stderr, autoErrStr)
		return fmt.Errorf("failed to merge PR #%s: %w", pr, err)
	}

	g.out.Successf("Auto-merge enabled for PR #%s", pr)
	return nil
}

func mergePRImmediate(out *output.Client, pr, repo string) error {
	immediateArgs := []string{"pr", "merge", pr, "--merge", "--delete-branch"}
	if repo != "" {
		immediateArgs = append(immediateArgs, "--repo", repo)
	}
	for attempt := range 10 {
		var immediateOut bytes.Buffer
		immediateCmd := exec.Command("gh", immediateArgs...)
		immediateCmd.Stdout = os.Stdout
		immediateCmd.Stderr = &immediateOut
		if err := immediateCmd.Run(); err != nil {
			errStr := immediateOut.String()
			if strings.Contains(errStr, "not mergeable") {
				out.Debugf("PR #%s not mergeable yet, retrying (attempt %d/10)...", pr, attempt+1)
				time.Sleep(5 * time.Second)
				continue
			}
			fmt.Fprint(os.Stderr, errStr)
			return fmt.Errorf("failed to merge PR #%s: %w", pr, err)
		}
		out.Successf("Merged PR #%s", pr)
		return nil
	}
	return fmt.Errorf("PR #%s was not mergeable after 10 attempts", pr)
}
