package workflow

import (
	"fmt"
	"os"
	"os/exec"
)

// FollowLogs runs `argo logs -f` for the given workflow and streams output to stdout/stderr.
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
