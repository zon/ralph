package git

import "os/exec"

// Installed reports whether the git CLI is on the PATH.
func Installed() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// InsideRepository reports whether the current directory is inside a git
// repository.
func InsideRepository() bool {
	_, err := FindRepoRoot()
	return err == nil
}

// ConfigGet returns the effective value of key, or "" when key is not set in
// any of the local, global, or system configuration files.
func ConfigGet(key string) string {
	value, err := runGit("config", "--get", key)
	if err != nil {
		return ""
	}
	return value
}
