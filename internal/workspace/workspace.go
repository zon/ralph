package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/output"
)

const (
	DefaultOpenCodeSecretsDir = "/secrets/opencode"
	DefaultWorkspaceDir       = "/workspace"
	DefaultWorkDir            = "/workspace/repo"
)

// SetupOpenCodeCredentials loads the OpenCode credentials mounted at
// DefaultOpenCodeSecretsDir into OpenCode's credential store. A Secret without
// a credentials document places nothing.
func SetupOpenCodeCredentials(out *output.Client) error {
	return setupOpenCodeCredentials(DefaultOpenCodeSecretsDir, out, opencode.ImportCredentials)
}

func setupOpenCodeCredentials(secretsDir string, out *output.Client, importCredentials func(path string) error) error {
	out.Info("Setting up OpenCode credentials...")

	credentialsFile := filepath.Join(secretsDir, "auth.json")
	switch _, err := os.Stat(credentialsFile); {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("failed to check OpenCode credentials: %w", err)
	}

	if err := importCredentials(credentialsFile); err != nil {
		return fmt.Errorf("failed to import OpenCode credentials: %w", err)
	}
	out.Infof("Imported OpenCode credentials from %s", credentialsFile)
	return nil
}

func PrepareWorkspace(out *output.Client, repoURL, branch, workDir string) error {
	out.Infof("Cloning repository: %s", repoURL)

	if err := os.MkdirAll(filepath.Dir(workDir), 0755); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}

	if _, err := os.Stat(workDir); err == nil {
		os.RemoveAll(workDir)
	}

	if err := git.Clone(repoURL, branch, workDir); err != nil {
		if err := git.Clone(repoURL, "", workDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	if err := Chdir(workDir); err != nil {
		return fmt.Errorf("failed to change to work dir: %w", err)
	}

	return nil
}

// Chdir changes the process working directory and keeps PWD in step. The
// OpenCode CLI resolves its project directory from PWD rather than the real
// working directory, so a bare os.Chdir would leave a worktree run pointed at
// the checkout the run started from.
func Chdir(dir string) error {
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", dir, err)
	}
	if wd, err := os.Getwd(); err == nil {
		_ = os.Setenv("PWD", wd)
	}
	return nil
}

func SetupSymlinks(out *output.Client) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	for _, cm := range cfg.Workflow.ConfigMaps {
		if cm.Link {
			if err := link(cwd, DefaultWorkspaceDir, cm.DestFile, cm.DestDir, out); err != nil {
				return err
			}
		}
	}

	for _, secret := range cfg.Workflow.Secrets {
		if secret.Link {
			if err := link(cwd, DefaultWorkspaceDir, secret.DestFile, secret.DestDir, out); err != nil {
				return err
			}
		}
	}

	return nil
}

func link(cwd, workspaceDir, destFile, destDir string, out *output.Client) error {
	dest := destFile
	if dest == "" {
		dest = destDir
	}
	if dest == "" {
		return nil
	}

	var src string
	if filepath.IsAbs(dest) {
		src = dest
	} else {
		src = filepath.Join(workspaceDir, dest)
	}
	linkPath := filepath.Join(cwd, dest)

	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("failed to stat source %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", linkPath, err)
	}

	if _, err := os.Lstat(linkPath); err == nil {
		return nil
	}

	out.Infof("Linking %s -> %s", linkPath, src)
	if err := os.Symlink(src, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, src, err)
	}
	return nil
}
