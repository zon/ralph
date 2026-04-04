package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("git %v failed: %w (output: %s)", args, err, output)
	}
	return strings.TrimSpace(string(output)), nil
}
