package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

const (
	DefaultOpenCodeSecretsDir = "/secrets/opencode"
	DefaultWorkspaceDir       = "/workspace"
	DefaultWorkDir            = "/workspace/repo"
)

func SetupOpenCodeCredentials() error {
	logger.Info("Setting up OpenCode credentials...")

	openCodeDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "opencode")
	if err := os.MkdirAll(openCodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create OpenCode directory: %w", err)
	}

	authFile := filepath.Join(DefaultOpenCodeSecretsDir, "auth.json")
	if _, err := os.Stat(authFile); err == nil {
		destPath := filepath.Join(openCodeDir, "auth.json")
		data, err := os.ReadFile(authFile)
		if err != nil {
			return fmt.Errorf("failed to read auth file: %w", err)
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write auth file: %w", err)
		}
		logger.Infof("Copied OpenCode credentials to %s", destPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check auth file: %w", err)
	}

	return nil
}

func PrepareWorkspace(repoURL, branch, workDir string) error {
	logger.Infof("Cloning repository: %s", repoURL)

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

func Chdir(dir string) error {
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", dir, err)
	}
	return nil
}
