package argo

import (
	"bytes"
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
	ListWorkflowNames(ctx K8sContext) ([]string, error)
	StopWorkflow(ctx K8sContext, workflowName string) error
	Logs(ctx K8sContext, workflowName string, follow bool) error
	SubmitYAML(ctx context.Context, workflowYAML string, kubeCtx K8sContext) (string, error)
}

type client struct{}

var _ Client = (*client)(nil)

func NewClient() Client {
	return &client{}
}

func (c *client) ListWorkflows(ctx K8sContext) error {
	output, err := runList(ctx)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, output)
	return nil
}

func (c *client) ListWorkflowNames(ctx K8sContext) ([]string, error) {
	output, err := runList(ctx)
	if err != nil {
		return nil, err
	}
	return parseWorkflowNames(output), nil
}

func listArgs(ctx K8sContext) []string {
	args := []string{"list", "-n", ctx.Namespace, "-l", "app.kubernetes.io/managed-by=ralph"}
	if ctx.Name != "" {
		args = append(args, "--context", ctx.Name)
	}
	return args
}

func runList(ctx K8sContext) (string, error) {
	cmd := exec.Command("argo", listArgs(ctx)...)
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to list workflows: %w", err)
	}

	return buffer.String(), nil
}

func parseWorkflowNames(output string) []string {
	var names []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if strings.ToUpper(fields[0]) == "NAME" {
			continue
		}
		names = append(names, fields[0])
	}
	return names
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

func logsArgs(ctx K8sContext, workflowName string, follow bool) []string {
	args := []string{"logs", "-n", ctx.Namespace}
	if follow {
		args = append(args, "-f")
	}
	if ctx.Name != "" {
		args = append(args, "--context", ctx.Name)
	}
	return append(args, workflowName)
}

func (c *client) Logs(ctx K8sContext, workflowName string, follow bool) error {
	cmd := exec.Command("argo", logsArgs(ctx, workflowName, follow)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get logs for workflow %s: %w", workflowName, err)
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

	workflowName := extractWorkflowName(string(output))
	if workflowName == "" {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			workflowName = strings.TrimSpace(lines[0])
		}
	}
	return workflowName, nil
}

func extractWorkflowName(output string) string {
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
