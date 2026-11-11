package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/output"
)

// ErrNoCommitsBetweenBranches is returned when gh pr create fails because the
// head branch has no commits ahead of the base branch. This is not an error in
// the traditional sense. It means the work was already complete before this
// run started, so there is nothing to open a PR for.
var ErrNoCommitsBetweenBranches = errors.New("no commits between branches")

// PullRequest is one open pull request returned by the GitHub CLI.
type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
}

// GHClient is the interface for GitHub CLI operations.
type GHClient interface {
	IsReady() bool
	FindExistingPR(head string) (string, error)
	CreatePR(title, body, base, head string) (string, error)
	ListCollaborators(ctx context.Context, owner, repo string) ([]string, error)
}

// OpenPullRequestLister lists the repository's open pull requests across every
// base branch. GHClient implementations that support prefix prompt checks
// implement it.
type OpenPullRequestLister interface {
	ListOpenPullRequests() ([]PullRequest, error)
}

// GH implements GHClient by shelling out to the gh CLI.
type GH struct {
	out *output.Client
}

func NewGH(out *output.Client) *GH {
	return &GH{out: out}
}

// GHInstalled reports whether the gh CLI is on the PATH.
func GHInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// GHCliToken returns the token stored by the gh CLI login, or an empty string
// when gh is not installed or not authenticated.
func GHCliToken() string {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

// ListOpenPullRequests lists the repository's open pull requests on every base
// branch. The caller filters them by head branch.
func (g *GH) ListOpenPullRequests() ([]PullRequest, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--state", "open",
		"--json", "number,url,title,headRefName,baseRefName",
		"--limit", "1000",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list open pull requests: %w (stderr: %s)", err, stderr.String())
	}

	var prs []PullRequest
	if err := json.Unmarshal(stdout.Bytes(), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse open pull requests: %w", err)
	}
	return prs, nil
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
