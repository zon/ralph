package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	versionFilePath   = "internal/version/VERSION"
	chartYAMLFilePath = "charts/ralph-webhook/Chart.yaml"
)

func BumpPatch() error {
	currentVersion, err := ReadVersionFile()
	if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	newVersion := bumpPatch(currentVersion)

	if err := WriteVersionFile(newVersion); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	if err := UpdateChartYAML(newVersion); err != nil {
		return fmt.Errorf("failed to update chart YAML: %w", err)
	}

	return nil
}

func ReadVersionFile() (string, error) {
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func WriteVersionFile(version string) error {
	dir := filepath.Dir(versionFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(versionFilePath, []byte(version+"\n"), 0644)
}

func bumpPatch(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return version
	}

	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
}

func UpdateChartYAML(appVersion string) error {
	data, err := os.ReadFile(chartYAMLFilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	currentChartVersion := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			parts := strings.TrimSpace(strings.TrimPrefix(line, "version:"))
			currentChartVersion = strings.TrimSpace(parts)
			newChartVersion := bumpPatch(currentChartVersion)
			newLines = append(newLines, fmt.Sprintf("version: %s", newChartVersion))
		} else if strings.HasPrefix(line, "appVersion:") {
			newLines = append(newLines, fmt.Sprintf("appVersion: \"%s\"", appVersion))
		} else {
			newLines = append(newLines, line)
		}
	}

	if currentChartVersion == "" {
		return fmt.Errorf("version field not found in Chart.yaml")
	}

	return os.WriteFile(chartYAMLFilePath, []byte(strings.Join(newLines, "\n")), 0644)
}
