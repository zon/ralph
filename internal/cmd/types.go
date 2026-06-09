package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

type ExecutionSetup struct {
	ProjectFile   string
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
}

type CommandSetup struct {
	Command []string
	Config  *config.RalphConfig
}

func runCommand(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("command required")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
