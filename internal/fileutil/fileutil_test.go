package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOSFileSystem_ReadFile(t *testing.T) {
	fs := OSFileSystem{}
	tmpfile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello world")
	err := os.WriteFile(tmpfile, content, 0644)
	require.NoError(t, err)

	readContent, err := fs.ReadFile(tmpfile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestOSFileSystem_WriteFile(t *testing.T) {
	fs := OSFileSystem{}
	tmpfile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello world")
	err := fs.WriteFile(tmpfile, content, 0644)
	require.NoError(t, err)

	readContent, err := os.ReadFile(tmpfile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestOSFileSystem_Stat(t *testing.T) {
	fs := OSFileSystem{}
	tmpfile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello world")
	err := os.WriteFile(tmpfile, content, 0644)
	require.NoError(t, err)

	info, err := fs.Stat(tmpfile)
	require.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(len(content)), info.Size())
}

func TestOSFileSystem_Join(t *testing.T) {
	fs := OSFileSystem{}
	path := fs.Join("dir1", "dir2", "file.txt")
	assert.Equal(t, filepath.Join("dir1", "dir2", "file.txt"), path)
}

func TestOSFileSystem_MkdirAll(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")
	err := MkdirAll(newDir, 0755)
	require.NoError(t, err)

	info, err := os.Stat(newDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestOSFileSystem_Abs(t *testing.T) {
	fs := OSFileSystem{}
	// Create a dummy file to ensure the path exists for Abs to resolve correctly
	tmpFile := filepath.Join(t.TempDir(), "dummy.txt")
	err := os.WriteFile(tmpFile, []byte("dummy"), 0644)
	require.NoError(t, err)

	absPath, err := fs.Abs(tmpFile)
	require.NoError(t, err)
	expectedAbsPath, err := filepath.Abs(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, expectedAbsPath, absPath)
}

func TestOSFileSystem_Base(t *testing.T) {
	fs := OSFileSystem{}
	path := "/a/b/c.txt"
	base := fs.Base(path)
	assert.Equal(t, "c.txt", base)
}

func TestOSFileSystem_Ext(t *testing.T) {
	fs := OSFileSystem{}
	path := "/a/b/c.txt"
	ext := fs.Ext(path)
	assert.Equal(t, ".txt", ext)
}

func TestOSFileSystem_WalkDir(t *testing.T) {
	fs := OSFileSystem{}
	tmpDir := t.TempDir()

	// Create a directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "dir1", "subdir1"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dir1", "file2.yaml"), []byte("content"), 0644))

	var visitedPaths []string
	err := fs.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(tmpDir, path)
		visitedPaths = append(visitedPaths, relPath)
		return nil
	})
	require.NoError(t, err)

	expectedPaths := []string{".", "dir1", "dir1/file2.yaml", "dir1/subdir1", "file1.txt"}
	assert.ElementsMatch(t, expectedPaths, visitedPaths)
}

func TestOSFileSystem_Remove(t *testing.T) {
	fs := OSFileSystem{}
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("content"), 0644)
	require.NoError(t, err)

	err = fs.Remove(tmpFile)
	require.NoError(t, err)

	_, err = os.Stat(tmpFile)
	assert.True(t, os.IsNotExist(err))
}

func TestOSFileSystem_Rename(t *testing.T) {
	fs := OSFileSystem{}
	oldFile := filepath.Join(t.TempDir(), "old.txt")
	newFile := filepath.Join(t.TempDir(), "new.txt")
	err := os.WriteFile(oldFile, []byte("content"), 0644)
	require.NoError(t, err)

	err = fs.Rename(oldFile, newFile)
	require.NoError(t, err)

	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(newFile)
	assert.NoError(t, err)
}
