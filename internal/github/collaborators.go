package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func ListCollaborators(ctx context.Context, owner, repo string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s/collaborators", owner, repo),
		"--jq", ".[].login",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list collaborators for %s/%s: %w (stderr: %s)",
			owner, repo, err, stderr.String())
	}

	var logins []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			logins = append(logins, line)
		}
	}
	return logins, nil
}
