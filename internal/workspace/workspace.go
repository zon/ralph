package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/output"
)

const (
	DefaultOpenCodeSecretsDir = "/secrets/opencode"
	DefaultWorkspaceDir       = "/workspace"
	DefaultWorkDir            = "/workspace/repo"
)

func ReadOpenCodeCredentials(authFilePath string) ([]byte, error) {
	authFileContent, err := os.ReadFile(authFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("OpenCode auth.json not found at %s\n\nPlease ensure OpenCode is configured and the auth.json file exists.", authFilePath)
		}
		return nil, fmt.Errorf("failed to read auth.json: %w", err)
	}

	if len(authFileContent) == 0 {
		return nil, fmt.Errorf("auth.json is empty at %s", authFilePath)
	}

	return authFileContent, nil
}

func SetupOpenCodeCredentials(out *output.Client) error {
	out.Info("Setting up OpenCode credentials...")

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
		out.Infof("Copied OpenCode credentials to %s", destPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check auth file: %w", err)
	}

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

func Chdir(dir string) error {
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", dir, err)
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
