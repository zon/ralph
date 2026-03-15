package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// deleteFile removes a file from the filesystem and stages the deletion
func deleteFile(filePath string) error {
	// Remove the file from filesystem
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file '%s': %w", filePath, err)
	}

	// Stage the deletion
	cmd := exec.Command("git", "rm", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage deletion of '%s': %w (output: %s)", filePath, err, out.String())
	}

	return nil
}
