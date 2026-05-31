package auth

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const authFilePath = ".ralph/auth.yaml"

func Load() (map[string]string, error) {
	data, err := os.ReadFile(authFilePath)
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

func Write(keys map[string]string) error {
	dir := filepath.Dir(authFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(keys)
	if err != nil {
		return err
	}

	return os.WriteFile(authFilePath, data, 0644)
}
