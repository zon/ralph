package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
)

// SetupWorkspaceCmd creates symlinks for mounted config files/dirs into the working directory
type SetupWorkspaceCmd struct {
	WorkspaceDir string `help:"Workspace directory containing mounted config files" default:"/workspace"`
}

// Run executes the setup-workspace command (implements kong.Run interface)
func (s *SetupWorkspaceCmd) Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	for _, cm := range cfg.Workflow.ConfigMaps {
		if err := s.link(cwd, cm.DestFile, cm.DestDir); err != nil {
			return err
		}
	}

	for _, secret := range cfg.Workflow.Secrets {
		if err := s.link(cwd, secret.DestFile, secret.DestDir); err != nil {
			return err
		}
	}

	return nil
}

// link creates a symlink inside cwd pointing to the corresponding path under workspaceDir.
// destFile and destDir are the mount paths configured in .ralph/config.yaml.
func (s *SetupWorkspaceCmd) link(cwd, destFile, destDir string) error {
	dest := destFile
	if dest == "" {
		dest = destDir
	}
	if dest == "" {
		return nil
	}

	// The source is the same path but rooted at workspaceDir
	src := filepath.Join(s.WorkspaceDir, dest)

	// The symlink lives at dest relative to cwd
	linkPath := filepath.Join(cwd, dest)

	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", linkPath, err)
	}

	if _, err := os.Lstat(linkPath); err == nil {
		// Already exists, skip
		return nil
	}

	fmt.Printf("Linking %s -> %s\n", linkPath, src)
	return os.Symlink(src, linkPath)
}
