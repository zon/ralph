package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isGitRepository checks if the current directory is inside a git repository
func isGitRepository() bool {
	_, err := runGit("rev-parse", "--git-dir")
	return err == nil
}

// FindRepoRoot returns the root directory of the git repository
func FindRepoRoot() (string, error) {
	repoRoot, err := runGit("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to find repo root: %w", err)
	}

	if repoRoot == "" {
		return "", fmt.Errorf("failed to determine repo root")
	}

	return repoRoot, nil
}

// TmpPath returns a path under the repo root's tmp/ directory for the given filename,
// inserting the current PID before the extension (e.g. "foo.yaml" → "tmp/foo-<pid>.yaml").
// The tmp/ directory is created if it does not exist.
func TmpPath(name string) (string, error) {
	root, err := FindRepoRoot()
	if err != nil {
		return "", err
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp directory: %w", err)
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	pidName := fmt.Sprintf("%s-%d%s", base, os.Getpid(), ext)
	return filepath.Join(tmpDir, pidName), nil
}

// isDetachedHead checks if the repository is in a detached HEAD state
func isDetachedHead() (bool, error) {
	_, err := runGit("symbolic-ref", "-q", "HEAD")
	return err != nil, nil
}

// RevParse executes git rev-parse with the given arguments
func RevParse(args ...string) (string, error) {
	fullArgs := append([]string{"rev-parse"}, args...)
	out, err := runGit(fullArgs...)
	if err != nil {
		return "", fmt.Errorf("rev-parse failed: %w", err)
	}
	return strings.TrimSpace(out), nil
}
