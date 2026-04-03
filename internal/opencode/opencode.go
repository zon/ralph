package opencode

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// RunCommand executes the opencode command with the given model and prompt.
// It pipes stdout/stderr to the provided writers or os.Stdout/os.Stderr if nil.
func RunCommand(ctx context.Context, model, prompt string, stdoutWriter, stderrWriter io.Writer) error {
	args := []string{"run", "--model", model, prompt}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	if stdoutWriter != nil {
		cmd.Stdout = stdoutWriter
	} else {
		cmd.Stdout = os.Stdout
	}
	if stderrWriter != nil {
		cmd.Stderr = stderrWriter
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode command failed: %w", err)
	}
	return nil
}
