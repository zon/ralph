package versionbump

import (
	"fmt"
	"strconv"
	"strings"
)

func MinorBump(version string) (string, error) {
	major, minor, patch, err := parseSemver(version)
	if err != nil {
		return "", err
	}
	minor++
	patch = 0
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func PatchBump(version string) (string, error) {
	major, minor, patch, err := parseSemver(version)
	if err != nil {
		return "", err
	}
	patch++
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func parseSemver(version string) (major, minor, patch int, err error) {
	version = strings.TrimSpace(version)
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid semver format: %q", version)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %q", parts[0])
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %q", parts[1])
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %q", parts[2])
	}
	return major, minor, patch, nil
}

func AppBump(version string) (string, error) {
	return MinorBump(version)
}

func ChartBump(version string) (string, error) {
	return PatchBump(version)
}