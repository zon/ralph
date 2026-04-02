package argo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LookPath checks if the argo CLI is available on PATH.
func LookPath() error {
	_, err := exec.LookPath("argo")
	if err != nil {
		return fmt.Errorf("argo CLI not found - please install Argo CLI to use remote execution: https://github.com/argoproj/argo-workflows/releases")
	}
	return nil
}

// SubmitYAML submits a raw YAML string to Argo and returns the workflow name.
func SubmitYAML(yaml string, workflowContext string, namespace string) (string, error) {
	if err := LookPath(); err != nil {
		return "", err
	}

	args := []string{"submit", "-", "-n", namespace}
	if workflowContext != "" {
		args = append(args, "--context", workflowContext)
	}

	cmd := exec.CommandContext(context.Background(), "argo", args...)
	cmd.Stdin = strings.NewReader(yaml)

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

// ExtractWorkflowName extracts the workflow name from argo submit output.
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

// Stop stops a workflow by name.
func Stop(name, workflowContext, namespace string) error {
	if err := LookPath(); err != nil {
		return err
	}

	args := []string{"stop", "-n", namespace}
	if workflowContext != "" {
		args = append(args, "--context", workflowContext)
	}
	args = append(args, name)

	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// List lists workflows in a namespace.
func List(workflowContext, namespace string) error {
	if err := LookPath(); err != nil {
		return err
	}

	args := []string{"list", "-n", namespace}
	if workflowContext != "" {
		args = append(args, "--context", workflowContext)
	}

	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Logs streams logs of a workflow, optionally following.
func Logs(name, namespace string, follow bool, workflowContext string) error {
	if err := LookPath(); err != nil {
		return err
	}

	args := []string{"logs", "-n", namespace}
	if workflowContext != "" {
		args = append(args, "--context", workflowContext)
	}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, name)

	cmd := exec.Command("argo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
