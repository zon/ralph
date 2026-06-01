package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/config"
)

func setupIterationTestRepo(t *testing.T, hookContent string) string {
	t.Helper()

	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	workDir := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, out)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	readmePath := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial commit"},
		{"push", "origin", "HEAD"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	if hookContent != "" {
		hookPath := filepath.Join(remoteDir, "hooks", "pre-receive")
		if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
			t.Fatalf("failed to write hook: %v", err)
		}
	}

	ralphDir := filepath.Join(workDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("failed to create .ralph directory: %v", err)
	}
	repoConfig, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load repo config: %v", err)
	}
	configContent := "model: " + repoConfig.Model + "\n"
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create .ralph/config.yaml: %v", err)
	}

	for _, args := range [][]string{
		{"add", ".ralph"},
		{"commit", "-m", "add ralph config"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	return workDir
}
