package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectModifiedProjectFile(t *testing.T) {
	t.Run("returns empty when no project files exist", func(t *testing.T) {
		// Setup a fresh test repo for this subtest
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		// Initialize git repo
		cmd := exec.Command("git", "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		result, err := DetectModifiedProjectFile("projects")
		if err != nil {
			t.Fatalf("DetectModifiedProjectFile returned error: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})

	t.Run("detects new project file", func(t *testing.T) {
		// Setup a fresh test repo for this subtest
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		// Initialize git repo
		cmd := exec.Command("git", "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create projects directory and a new project file
		if err := os.MkdirAll("projects", 0755); err != nil {
			t.Fatalf("Failed to create projects dir: %v", err)
		}

		projectFile := filepath.Join("projects", "fix-ai-error-handling.yaml")
		projectContent := `name: fix-ai-error-handling
description: Fix AI error handling
requirements:
  - category: bug
    description: Fix the error
    passing: false
`
		if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
			t.Fatalf("Failed to write project file: %v", err)
		}

		result, err := DetectModifiedProjectFile("projects")
		if err != nil {
			t.Fatalf("DetectModifiedProjectFile returned error: %v", err)
		}

		expectedAbs, _ := filepath.Abs(projectFile)
		if result != expectedAbs {
			t.Errorf("Expected %s, got %s", expectedAbs, result)
		}
	})

	t.Run("detects modified project file", func(t *testing.T) {
		// Setup a fresh test repo for this subtest
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		// Initialize git repo
		cmd := exec.Command("git", "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Configure git user
		for _, args := range [][]string{
			{"config", "--local", "user.email", "test@example.com"},
			{"config", "--local", "user.name", "Test User"},
		} {
			c := exec.Command("git", args...)
			if out, err := c.CombinedOutput(); err != nil {
				t.Fatalf("git %v failed: %v\n%s", args, err, out)
			}
		}

		// Create projects directory and a project file
		if err := os.MkdirAll("projects", 0755); err != nil {
			t.Fatalf("Failed to create projects dir: %v", err)
		}

		projectFile := filepath.Join("projects", "test-project.yaml")
		projectContent := `name: test-project
description: Test project
requirements: []
`
		if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
			t.Fatalf("Failed to write project file: %v", err)
		}

		// Stage and commit the file first
		cmd = exec.Command("git", "add", "projects/test-project.yaml")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to stage project file: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "Add project file")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to commit project file: %v", err)
		}

		// Now modify the file
		modifiedContent := projectContent + "\n# modified\n"
		if err := os.WriteFile(projectFile, []byte(modifiedContent), 0644); err != nil {
			t.Fatalf("Failed to modify project file: %v", err)
		}

		result, err := DetectModifiedProjectFile("projects")
		if err != nil {
			t.Fatalf("DetectModifiedProjectFile returned error: %v", err)
		}

		expectedAbs, _ := filepath.Abs(projectFile)
		if result != expectedAbs {
			t.Errorf("Expected %s, got %s", expectedAbs, result)
		}
	})

	t.Run("ignores non-yaml files", func(t *testing.T) {
		// Setup a fresh test repo for this subtest
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		// Initialize git repo
		cmd := exec.Command("git", "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create projects directory with a non-yaml file
		if err := os.MkdirAll("projects", 0755); err != nil {
			t.Fatalf("Failed to create projects dir: %v", err)
		}

		readmeFile := filepath.Join("projects", "README.md")
		if err := os.WriteFile(readmeFile, []byte("# Projects\n"), 0644); err != nil {
			t.Fatalf("Failed to write README: %v", err)
		}

		result, err := DetectModifiedProjectFile("projects")
		if err != nil {
			t.Fatalf("DetectModifiedProjectFile returned error: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty string for non-yaml file, got %s", result)
		}
	})
}

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits) - using --local to ensure isolation
	cmd = exec.Command("git", "config", "--local", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "config", "--local", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir
}

// setupBareRemoteRepo creates a temporary bare remote and a clone of it,
// configures git identity, makes an initial commit, pushes it, and returns
// (workDir, remoteDir). The caller must chdir into workDir before calling any
// git functions that rely on the working directory.
func setupBareRemoteRepo(t *testing.T) (workDir, remoteDir string) {
	t.Helper()

	remoteDir = t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	workDir = t.TempDir()
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

	// Create and push an initial commit so HEAD exists on the remote.
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
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

	return workDir, remoteDir
}
