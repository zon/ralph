package git

import (
	"fmt"
)

// Config sets a git configuration value globally or locally
func Config(global bool, key, value string) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	_, err := runGit(args...)
	if err != nil {
		return fmt.Errorf("failed to set git config %s=%s: %w", key, value, err)
	}
	return nil
}
