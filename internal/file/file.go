package file

import (
	"io/fs"
	"os"
	"path/filepath"
)

// ReadFile reads the named file and returns its contents.
func ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// WriteFile writes data to the named file, creating it if necessary.
func WriteFile(name string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return err
	}
	return nil
}

// Stat returns a FileInfo describing the named file.
func Stat(name string) (fs.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// IsNotExist returns a boolean indicating whether the error is known to report
// that a file or directory does not exist.
func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// Remove removes the named file or directory.
func Remove(name string) error {
	return os.Remove(name)
}

// Rename renames (moves) oldpath to newpath.
func Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// Getwd returns a rooted path name corresponding to the current directory.
func Getwd() (string, error) {
	return os.Getwd()
}

// Abs returns an absolute representation of path.
func Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// Join joins any number of path elements into a single path.
func Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path.
func Base(path string) string {
	return filepath.Base(path)
}

// Ext returns the file name extension used by path.
func Ext(path string) string {
	return filepath.Ext(path)
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator.
func Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// IsAbs reports whether the path is absolute.
func IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// Dir returns all but the last element of path, typically the path's directory.
func Dir(path string) string {
	return filepath.Dir(path)
}

// WalkDir walks the file tree rooted at root, calling fn for each file or directory.
func WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}
