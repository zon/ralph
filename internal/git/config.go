package git

import (
	"fmt"
	"strings"
)

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

func ConfigList(global bool) (string, error) {
	args := []string{"config", "--list"}
	if global {
		args = append(args, "--global")
	}

	output, err := runGit(args...)
	if err != nil {
		return "", fmt.Errorf("failed to list git config: %w", err)
	}
	return output, nil
}

func ConfigUnset(global bool, key string) error {
	args := []string{"config", "--unset-all"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	_, err := runGit(args...)
	if err != nil {
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}

func ConfigGet(key string) (string, error) {
	output, err := runGit("config", "--get", key)
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	return strings.TrimSpace(output), nil
}
