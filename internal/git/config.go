package git

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Config sets a git configuration value globally or locally
func Config(global bool, key, value string) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git config %s=%s: %w (output: %s)", key, value, err, out.String())
	}
	return nil
}
