package argo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

func FollowLogs(namespace, workflowName, kubeContext string) error {
	args := []string{"logs", "-n", namespace, "-f", workflowName}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("argo logs failed: %w", err)
	}
	return nil
}

func SubmitYAML(workflowYAML string, workflowContext string, namespace string) (string, error) {
	if _, err := exec.LookPath("argo"); err != nil {
		return "", fmt.Errorf("argo CLI not found - please install Argo CLI to use remote execution: https://github.com/argoproj/argo-workflows/releases")
	}

	args := []string{"submit", "-", "-n", namespace}
	if workflowContext != "" {
		args = append(args, "--context", workflowContext)
	}

	cmd := exec.CommandContext(context.Background(), "argo", args...)
	cmd.Stdin = strings.NewReader(workflowYAML)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to submit workflow: %w\nOutput: %s", err, string(output))
	}

	workflowName := ExtractWorkflowName(string(output))
	if workflowName == "" {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			workflowName = strings.TrimSpace(lines[0])
		}
	}
	return workflowName, nil
}

func ExtractWorkflowName(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Name:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}
