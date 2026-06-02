package setup_workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
)

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type FsClient interface {
	Getwd() (string, error)
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Lstat(name string) (os.FileInfo, error)
	Symlink(oldName, newName string) error
}

type Logger interface {
	Infof(format string, args ...interface{})
}

type SetupWorkspaceCmd struct {
	workspaceDir string
	configLoader ConfigLoader
	fs           FsClient
	log          Logger
}

func New(workspaceDir string, configLoader ConfigLoader, fs FsClient, log Logger) *SetupWorkspaceCmd {
	return &SetupWorkspaceCmd{
		workspaceDir: workspaceDir,
		configLoader: configLoader,
		fs:           fs,
		log:          log,
	}
}

func (s *SetupWorkspaceCmd) Run() error {
	cfg, err := s.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := s.fs.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	for _, cm := range cfg.Workflow.ConfigMaps {
		if cm.Link {
			if err := s.link(cwd, cm.DestFile, cm.DestDir); err != nil {
				return err
			}
		}
	}

	for _, secret := range cfg.Workflow.Secrets {
		if secret.Link {
			if err := s.link(cwd, secret.DestFile, secret.DestDir); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SetupWorkspaceCmd) link(cwd, destFile, destDir string) error {
	dest := destFile
	if dest == "" {
		dest = destDir
	}
	if dest == "" {
		return nil
	}

	src := filepath.Join(s.workspaceDir, dest)
	linkPath := filepath.Join(cwd, dest)

	if _, err := s.fs.Stat(src); err != nil {
		return fmt.Errorf("failed to stat source %s: %w", src, err)
	}

	if err := s.fs.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", linkPath, err)
	}

	if _, err := s.fs.Lstat(linkPath); err == nil {
		return nil
	}

	s.log.Infof("Linking %s -> %s", linkPath, src)
	if err := s.fs.Symlink(src, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, src, err)
	}
	return nil
}
