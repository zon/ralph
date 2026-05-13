package version

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed VERSION
var versionFile string

func Version() string {
	return strings.TrimSpace(versionFile)
}

func BumpMinor() error {
	return BumpMinorToFile(versionFilePath())
}

func versionFilePath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "VERSION")
}

func BumpMinorToFile(path string) error {
	v := Version()
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version %q is not valid semver", v)
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return fmt.Errorf("version %q is not valid semver: %w", v, err)
	}
	minor++
	newVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	if err := os.WriteFile(path, []byte(newVersion+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write VERSION file: %w", err)
	}
	return nil
}

func BumpMinorString(v string) (string, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("version %q is not valid semver", v)
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return "", fmt.Errorf("version %q is not valid semver: %w", v, err)
	}
	minor++
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}
