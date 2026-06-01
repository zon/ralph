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

type Client interface {
	ListWorkflows(ctx K8sContext) error
	StopWorkflow(ctx K8sContext, workflowName string) error
	FollowLogs(ctx K8sContext, workflowName string) error
	SubmitYAML(ctx context.Context, workflowYAML string, kubeCtx K8sContext) (string, error)
}

type client struct{}

var _ Client = (*client)(nil)

func NewClient() Client {
	return &client{}
}

func (c *client) ListWorkflows(ctx K8sContext) error {
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

func (c *client) StopWorkflow(ctx K8sContext, workflowName string) error {
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

func (c *client) FollowLogs(ctx K8sContext, workflowName string) error {
	args := []string{"logs", "-n", ctx.Namespace, "-f", workflowName}
	if ctx.Name != "" {
		args = append(args, "--context", ctx.Name)
	}
	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("argo logs failed: %w", err)
	}
	return nil
}

func (c *client) SubmitYAML(ctx context.Context, workflowYAML string, kubeCtx K8sContext) (string, error) {
	if _, err := exec.LookPath("argo"); err != nil {
		return "", fmt.Errorf("argo CLI not found - please install Argo CLI to use remote execution: https://github.com/argoproj/argo-workflows/releases")
	}

	args := []string{"submit", "-", "-n", kubeCtx.Namespace}
	if kubeCtx.Name != "" {
		args = append(args, "--context", kubeCtx.Name)
	}

	cmd := exec.CommandContext(ctx, "argo", args...)
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

func ListWorkflows(ctx K8sContext) error {
	return new(client).ListWorkflows(ctx)
}

func StopWorkflow(ctx K8sContext, workflowName string) error {
	return new(client).StopWorkflow(ctx, workflowName)
}

func FollowLogs(namespace, workflowName, kubeContext string) error {
	return new(client).FollowLogs(K8sContext{
		Name:      kubeContext,
		Namespace: namespace,
	}, workflowName)
}

func SubmitYAML(workflowYAML string, workflowContext string, namespace string) (string, error) {
	return new(client).SubmitYAML(context.Background(), workflowYAML, K8sContext{
		Name:      workflowContext,
		Namespace: namespace,
	})
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
