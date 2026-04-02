package fileutil

import (
	"io/fs"
	"os"
	"path/filepath"
)

// ReadFile reads the named file and returns its contents.
func ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile writes data to the named file, creating it if necessary.
func WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Stat returns a FileInfo describing the named file.
func Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// Remove removes the named file or directory.
func Remove(name string) error {
	return os.Remove(name)
}

// Rename renames (moves) oldpath to newpath.
func Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// WalkDir walks the file tree rooted at root, calling fn for each file or directory.
func WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// Abs returns an absolute representation of path.
func Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// Join joins any number of path elements into a single path.
func Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Dir returns all but the last element of path, typically the path's directory.
func Dir(path string) string {
	return filepath.Dir(path)
}

// Base returns the last element of path.
func Base(path string) string {
	return filepath.Base(path)
}

// Ext returns the file name extension used by path.
func Ext(path string) string {
	return filepath.Ext(path)
}

// IsNotExist returns a boolean indicating whether the error is known to report that a file or directory does not exist.
func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
