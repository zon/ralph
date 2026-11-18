package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		// Arguments and output can carry credentials, such as the installation
		// token in an insteadOf URL, so redact them before building the error.
		return trimmed, fmt.Errorf("git %v failed: %w (output: %s)", redactCredentials(fmt.Sprintf("%v", args)), err, redactCredentials(trimmed))
	}
	return trimmed, nil
}
