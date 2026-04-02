package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(tmpFile, content, 0644))

	readContent, err := ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	_, err = ReadFile(filepath.Join(tmpDir, "nonexistent"))
	assert.Error(t, err)
	assert.True(t, IsNotExist(err))
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("foo bar")

	err := WriteFile(tmpFile, content, 0644)
	require.NoError(t, err)

	readContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestStat(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("x"), 0644))

	info, err := Stat(tmpFile)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
	assert.Equal(t, filepath.Base(tmpFile), info.Name())

	_, err = Stat(filepath.Join(tmpDir, "nonexistent"))
	assert.Error(t, err)
	assert.True(t, IsNotExist(err))
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("x"), 0644))

	_, err := os.Stat(tmpFile)
	require.NoError(t, err)

	err = Remove(tmpFile)
	require.NoError(t, err)

	_, err = os.Stat(tmpFile)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestGetwd(t *testing.T) {
	wd, err := Getwd()
	require.NoError(t, err)
	assert.NotEmpty(t, wd)
}

func TestAbs(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	abs, err := Abs(".")
	require.NoError(t, err)
	assert.Equal(t, wd, abs)

	abs, err = Abs("file.go")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(wd, "file.go"), abs)
}

func TestJoin(t *testing.T) {
	assert.Equal(t, "a/b/c", Join("a", "b", "c"))
	assert.Equal(t, "/a/b/c", Join("/a", "b", "c"))
}

func TestBase(t *testing.T) {
	assert.Equal(t, "file.txt", Base("/path/to/file.txt"))
	assert.Equal(t, "file.txt", Base("file.txt"))
}

func TestExt(t *testing.T) {
	assert.Equal(t, ".txt", Ext("file.txt"))
	assert.Equal(t, "", Ext("file"))
}

func TestRel(t *testing.T) {
	// Test case where relative path exists
	rel, err := Rel("/a/b", "/a/b/c/d")
	require.NoError(t, err)
	assert.Equal(t, "c/d", rel)

	// Test case where relative path does not exist - should error
	gotRel, gotErr := Rel("/a/b", "/x/y")
	expectedRel, expectedErr := filepath.Rel("/a/b", "/x/y")
	if expectedErr != nil {
		assert.Error(t, gotErr)
	} else {
		// unlikely, but if filepath.Rel succeeds, we should match
		assert.NoError(t, gotErr)
		assert.Equal(t, expectedRel, gotRel)
	}
}

func TestIsAbs(t *testing.T) {
	assert.True(t, IsAbs("/absolute/path"))
	assert.False(t, IsAbs("relative/path"))
}

func TestDir(t *testing.T) {
	assert.Equal(t, "/path/to", Dir("/path/to/file.txt"))
	assert.Equal(t, ".", Dir("file.txt"))
}
