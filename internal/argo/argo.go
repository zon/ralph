package argo

import (
	"fmt"
	"os"
	"os/exec"
)

type K8sContext struct {
	Name      string
	Namespace string
}

func ListWorkflows(ctx K8sContext) error {
	args := []string{"list", "-n", ctx.Namespace}
	if ctx.Name != "" {
		args = append(args, "--context", ctx.Name)
	}

	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	return nil
}

func StopWorkflow(ctx K8sContext, workflowName string) error {
	args := []string{"stop", "-n", ctx.Namespace}
	if ctx.Name != "" {
		args = append(args, "--context", ctx.Name)
	}
	args = append(args, workflowName)

	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop workflow: %w", err)
	}

	return nil
}
