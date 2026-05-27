package architecture

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SpecModule matches the module format in specs/architecture.yaml and feature architecture files.
type SpecModule struct {
	Path          string `yaml:"path"`
	Description   string `yaml:"description"`
	Orchestration bool   `yaml:"orchestration,omitempty"`
}

// SpecArchitecture is the structure of specs/architecture.yaml and feature architecture files.
type SpecArchitecture struct {
	Modules []SpecModule `yaml:"modules"`
}

// LoadSpec loads a spec-format architecture YAML. Returns nil, nil if the file does not exist.
func LoadSpec(path string) (*SpecArchitecture, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var arch SpecArchitecture
	if err := yaml.Unmarshal(data, &arch); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &arch, nil
}

// SaveSpec writes a spec-format architecture YAML with 2-space indentation.
func SaveSpec(path string, arch *SpecArchitecture) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(arch); err != nil {
		return fmt.Errorf("failed to encode architecture: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// MigrateImplementedModules moves modules from the feature architecture at featurePath
// into the global architecture at globalPath whenever the module's path exists under repoRoot.
// Modules that match an existing global entry by path replace it; others are appended.
// Returns the count of modules migrated.
func MigrateImplementedModules(featurePath, globalPath, repoRoot string) (int, error) {
	feature, err := LoadSpec(featurePath)
	if err != nil {
		return 0, fmt.Errorf("failed to load feature architecture: %w", err)
	}
	if feature == nil || len(feature.Modules) == 0 {
		return 0, nil
	}

	global, err := LoadSpec(globalPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load global architecture: %w", err)
	}
	if global == nil {
		global = &SpecArchitecture{}
	}

	var remaining []SpecModule
	migrated := 0

	for _, mod := range feature.Modules {
		if _, statErr := os.Stat(filepath.Join(repoRoot, mod.Path)); os.IsNotExist(statErr) {
			remaining = append(remaining, mod)
			continue
		}

		replaced := false
		for i := range global.Modules {
			if global.Modules[i].Path == mod.Path {
				global.Modules[i] = mod
				replaced = true
				break
			}
		}
		if !replaced {
			global.Modules = append(global.Modules, mod)
		}
		migrated++
	}

	if migrated == 0 {
		return 0, nil
	}

	if err := SaveSpec(globalPath, global); err != nil {
		return 0, fmt.Errorf("failed to save global architecture: %w", err)
	}

	feature.Modules = remaining
	if len(feature.Modules) == 0 {
		if err := os.Remove(featurePath); err != nil && !os.IsNotExist(err) {
			return migrated, fmt.Errorf("failed to remove feature architecture: %w", err)
		}
	} else {
		if err := SaveSpec(featurePath, feature); err != nil {
			return migrated, fmt.Errorf("failed to save feature architecture: %w", err)
		}
	}

	return migrated, nil
}
