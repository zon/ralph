package version

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var versionFilePath = "internal/version/VERSION"

func BumpPatch() error {
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	version := strings.TrimSpace(string(data))
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid version format: %s", version)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("failed to parse patch version: %w", err)
	}

	newVersion := fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)

	if err := os.WriteFile(versionFilePath, []byte(newVersion+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}
