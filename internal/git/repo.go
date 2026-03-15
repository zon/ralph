package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// IsGitRepository checks if the current directory is inside a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// FindRepoRoot returns the root directory of the git repository
func FindRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to find repo root: %w (output: %s)", err, out.String())
	}

	repoRoot := strings.TrimSpace(out.String())
	if repoRoot == "" {
		return "", fmt.Errorf("failed to determine repo root")
	}

	return repoRoot, nil
}

// IsDetachedHead checks if the repository is in a detached HEAD state
func IsDetachedHead() (bool, error) {
	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	err := cmd.Run()

	// Exit code 0 = on a branch (not detached)
	// Exit code 1 = detached HEAD
	isDetached := err != nil

	return isDetached, nil
}

// RevParse executes git rev-parse with the given arguments
func RevParse(args ...string) (string, error) {
	fullArgs := append([]string{"rev-parse"}, args...)
	cmd := exec.Command("git", fullArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("rev-parse failed: %w (output: %s)", err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}
