package k8s

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// runKubectl runs a kubectl command with the provided arguments and stdin.
func runKubectl(ctx context.Context, stdin *bytes.Buffer, args ...string) (*bytes.Buffer, error) {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return nil, fmt.Errorf("kubectl not found in PATH - please install kubectl")
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if stdin != nil {
		cmd.Stdin = stdin
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("kubectl command failed: %w (stderr: %s)", err, stderr.String())
	}

	return &stdout, nil
}
