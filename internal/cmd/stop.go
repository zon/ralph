package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/zon/ralph/internal/config"
)

type StopCmd struct {
	WorkflowName string `arg:"" help:"Name of the workflow to stop"`
	Context      string `help:"Kubernetes context to use" name:"context" optional:""`
}

func (s *StopCmd) Run() error {
	ctx := context.Background()

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := resolveKubeContext(ctx, ralphConfig, s.Context, "")
	if err != nil {
		return err
	}

	args := []string{"stop", "-n", k8sCtx.Namespace}
	if k8sCtx.Name != "" {
		args = append(args, "--context", k8sCtx.Name)
	}
	args = append(args, s.WorkflowName)

	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop workflow: %w", err)
	}

	return nil
}
