package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
)

// SetupWorkspaceCmd copies mounted config files/dirs into the working directory
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
		if err := s.copyEntry(cwd, cm.DestFile, cm.DestDir); err != nil {
			return err
		}
	}

	for _, secret := range cfg.Workflow.Secrets {
		if err := s.copyEntry(cwd, secret.DestFile, secret.DestDir); err != nil {
			return err
		}
	}

	return nil
}

// copyEntry copies a file or directory from workspaceDir into cwd.
// destFile and destDir are the mount paths configured in .ralph/config.yaml.
func (s *SetupWorkspaceCmd) copyEntry(cwd, destFile, destDir string) error {
	dest := destFile
	if dest == "" {
		dest = destDir
	}
	if dest == "" {
		return nil
	}

	src := filepath.Join(s.WorkspaceDir, dest)
	dstPath := filepath.Join(cwd, dest)

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source %s: %w", src, err)
	}

	if info.IsDir() {
		fmt.Printf("Copying dir %s -> %s\n", src, dstPath)
		return copyDir(src, dstPath)
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", dstPath, err)
	}
	fmt.Printf("Copying %s -> %s\n", src, dstPath)
	return copyFile(src, dstPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", src, err)
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", src, err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
	}
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
