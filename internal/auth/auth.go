package auth

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func authFilePath(rootDir string) string {
	return filepath.Join(rootDir, ".ralph", "auth.yaml")
}

func Load(rootDir string) (map[string]string, error) {
	data, err := os.ReadFile(authFilePath(rootDir))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	var keys map[string]string
	if err := yaml.Unmarshal(data, &keys); err != nil {
		return nil, err
	}

	if keys == nil {
		return map[string]string{}, nil
	}

	return keys, nil
}

func Write(rootDir string, keys map[string]string) error {
	dir := filepath.Dir(authFilePath(rootDir))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(keys)
	if err != nil {
		return err
	}

	return os.WriteFile(authFilePath(rootDir), data, 0644)
}
