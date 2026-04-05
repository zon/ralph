package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCleanWorkingTree(t *testing.T) {
	t.Run("returns nil when working tree is clean", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := EnsureCleanWorkingTree()
		require.NoError(t, err, "EnsureCleanWorkingTree should return nil for clean tree")
	})

	t.Run("returns error when unstaged changes exist", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		testFile := filepath.Join(tempDir, "modified.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

		err := EnsureCleanWorkingTree()
		require.Error(t, err, "EnsureCleanWorkingTree should return error for dirty tree")
		assert.ErrorIs(t, err, ErrUncommittedChanges)
	})

	t.Run("returns error when staged changes exist", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		testFile := filepath.Join(tempDir, "staged.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))
		require.NoError(t, StageFile("staged.txt"))

		err := EnsureCleanWorkingTree()
		require.Error(t, err, "EnsureCleanWorkingTree should return error for dirty tree")
		assert.ErrorIs(t, err, ErrUncommittedChanges)
	})
}

func TestCommitWithVerification(t *testing.T) {
	t.Run("creates commit and verifies it exists", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		testFile := filepath.Join(tempDir, "verify.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))
		require.NoError(t, StageFile("verify.txt"))

		err := CommitWithVerification("add verify file")
		require.NoError(t, err, "CommitWithVerification failed")

		commitHash, err := RevParse("HEAD")
		require.NoError(t, err, "Failed to get commit hash")
		assert.NotEmpty(t, commitHash, "Commit hash should not be empty")
	})

	t.Run("fails when no staged changes", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := CommitWithVerification("empty commit")
		require.Error(t, err, "CommitWithVerification should fail with no staged changes")
	})
}

func TestAtomicCommitWithFiles(t *testing.T) {
	t.Run("stages files, commits, and verifies", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		file1 := filepath.Join(tempDir, "atomic1.txt")
		file2 := filepath.Join(tempDir, "atomic2.txt")
		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(file2, []byte("content2"), 0644))

		err := AtomicCommitWithFiles("add atomic files", []string{"atomic1.txt", "atomic2.txt"})
		require.NoError(t, err, "AtomicCommitWithFiles failed")

		commitHash, err := RevParse("HEAD")
		require.NoError(t, err, "Failed to verify commit")
		assert.NotEmpty(t, commitHash, "Commit hash should not be empty")
	})

	t.Run("fails when file list is empty", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := AtomicCommitWithFiles("no files", []string{})
		require.Error(t, err, "AtomicCommitWithFiles should fail with empty file list")
	})

	t.Run("fails when file does not exist", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := AtomicCommitWithFiles("nonexistent file", []string{"nonexistent.txt"})
		require.Error(t, err, "AtomicCommitWithFiles should fail with nonexistent file")
	})
}

func TestFetchAndRebase(t *testing.T) {
	t.Run("fetches and rebases successfully", func(t *testing.T) {
		workDir, _ := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		err := FetchAndRebase(nil, "main")
		require.NoError(t, err, "FetchAndRebase failed")
	})

	t.Run("fetches when branch has new remote commits", func(t *testing.T) {
		remoteDir := t.TempDir()
		require.NoError(t, runCommand(remoteDir, "git", "init", "--bare"))

		workDir1 := t.TempDir()
		require.NoError(t, runCommand(workDir1, "git", "clone", remoteDir, "."))
		configureGitUser(t, workDir1)

		branchName := "rebase-test"
		runCommand(workDir1, "git", "checkout", "-b", branchName)
		writeFile(workDir1, "test1.txt", "test1")
		runCommand(workDir1, "git", "add", ".")
		runCommand(workDir1, "git", "commit", "-m", "commit 1")
		runCommand(workDir1, "git", "push", "origin", branchName)

		workDir2 := t.TempDir()
		require.NoError(t, runCommand(workDir2, "git", "clone", remoteDir, "."))
		configureGitUser(t, workDir2)
		runCommand(workDir2, "git", "checkout", branchName)

		writeFile(workDir1, "test2.txt", "test2")
		runCommand(workDir1, "git", "add", ".")
		runCommand(workDir1, "git", "commit", "-m", "commit 2")
		runCommand(workDir1, "git", "push", "origin", branchName)

		t.Chdir(workDir2)
		err := FetchAndRebase(nil, branchName)
		require.NoError(t, err, "FetchAndRebase failed")
	})
}

func TestGetStagedFiles(t *testing.T) {
	t.Run("returns empty when no staged files", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		files, err := GetStagedFiles()
		require.NoError(t, err, "GetStagedFiles failed")
		assert.Empty(t, files, "Expected empty file list for clean staging area")
	})

	t.Run("returns staged files", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		writeFile(tempDir, "staged1.txt", "content1")
		writeFile(tempDir, "staged2.txt", "content2")
		require.NoError(t, StageFile("staged1.txt"))
		require.NoError(t, StageFile("staged2.txt"))

		files, err := GetStagedFiles()
		require.NoError(t, err, "GetStagedFiles failed")
		assert.Len(t, files, 2, "Expected 2 staged files")
		assert.Contains(t, files, "staged1.txt")
		assert.Contains(t, files, "staged2.txt")
	})
}

func TestSyncBranch(t *testing.T) {
	t.Run("syncs branch with remote", func(t *testing.T) {
		workDir, _ := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		branchName := "synced-branch"
		runCommand(workDir, "git", "checkout", "-b", branchName)
		writeFile(workDir, "sync.txt", "content")
		runCommand(workDir, "git", "add", ".")
		runCommand(workDir, "git", "commit", "-m", "add sync file")
		runCommand(workDir, "git", "push", "origin", branchName)

		err := SyncBranch(nil, branchName)
		require.NoError(t, err, "SyncBranch failed")
	})
}

func TestSyncBranchWithVerification(t *testing.T) {
	t.Run("syncs and pushes successfully", func(t *testing.T) {
		workDir, _ := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		branchName := "push-verify-branch"
		runCommand(workDir, "git", "checkout", "-b", branchName)
		writeFile(workDir, "push.txt", "content")
		runCommand(workDir, "git", "add", ".")
		runCommand(workDir, "git", "commit", "-m", "add push file")
		runCommand(workDir, "git", "push", "origin", branchName)

		remoteURL, err := SyncBranchWithVerification(nil, branchName)
		require.NoError(t, err, "SyncBranchWithVerification failed")
		assert.NotEmpty(t, remoteURL, "Remote URL should not be empty")
	})
}

func TestValidateCleanTreeAndCommit(t *testing.T) {
	t.Run("commits files when tree is clean", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		writeFile(tempDir, "clean1.txt", "content1")
		writeFile(tempDir, "clean2.txt", "content2")

		err := AtomicCommitWithFiles("add clean files", []string{"clean1.txt", "clean2.txt"})
		require.NoError(t, err, "AtomicCommitWithFiles failed in clean tree")

		commitHash, err := RevParse("HEAD")
		require.NoError(t, err, "Failed to verify commit")
		assert.NotEmpty(t, commitHash)
	})

	t.Run("fails when tree is dirty", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		writeFile(tempDir, "dirty.txt", "dirty content")
		writeFile(tempDir, "clean.txt", "clean content")

		err := ValidateCleanTreeAndCommit("should fail", []string{"clean.txt"})
		require.Error(t, err, "ValidateCleanTreeAndCommit should fail with dirty tree")
		assert.ErrorIs(t, err, ErrUncommittedChanges)
	})
}

func writeFile(dir, name, content string) {
	os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %v (output: %s)", err, output)
	}
	return nil
}

func configureGitUser(t *testing.T, dir string) {
	t.Helper()
	runCommand(dir, "git", "config", "--local", "user.email", "test@example.com")
	runCommand(dir, "git", "config", "--local", "user.name", "Test User")
}
