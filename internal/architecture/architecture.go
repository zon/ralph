package architecture

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Architecture struct {
	Apps    []App    `yaml:"apps"`
	Modules []Module `yaml:"modules"`
}

type App struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Main        MainFunc  `yaml:"main"`
	Features    []Feature `yaml:"features"`
}

type MainFunc struct {
	File     string `yaml:"file"`
	Function string `yaml:"function"`
}

type Feature struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Functions   []FuncRef `yaml:"functions"`
}

type FuncRef struct {
	File string `yaml:"file"`
	Name string `yaml:"name"`
}

type Module struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

func (a *Architecture) Validate() []string {
	var errors []string

	for i, app := range a.Apps {
		if app.Name == "" {
			errors = append(errors, fmt.Sprintf("app[%d]: name is required", i))
		}
		if app.Description == "" {
			errors = append(errors, fmt.Sprintf("app[%d] (%s): description is required", i, app.Name))
		}
		if app.Main.File == "" {
			errors = append(errors, fmt.Sprintf("app[%d] (%s): main.file is required", i, app.Name))
		}
		if app.Main.Function == "" {
			errors = append(errors, fmt.Sprintf("app[%d] (%s): main.function is required", i, app.Name))
		}
		if len(app.Features) == 0 {
			errors = append(errors, fmt.Sprintf("app[%d] (%s): at least one feature is required", i, app.Name))
		}

		for j, feature := range app.Features {
			if feature.Name == "" {
				errors = append(errors, fmt.Sprintf("app[%d] (%s) feature[%d]: name is required", i, app.Name, j))
			}
			if feature.Description == "" {
				errors = append(errors, fmt.Sprintf("app[%d] (%s) feature[%d] (%s): description is required", i, app.Name, j, feature.Name))
			}
			if len(feature.Functions) == 0 {
				errors = append(errors, fmt.Sprintf("app[%d] (%s) feature[%d] (%s): at least one function ref is required", i, app.Name, j, feature.Name))
			}

			for k, fn := range feature.Functions {
				if fn.File == "" {
					errors = append(errors, fmt.Sprintf("app[%d] (%s) feature[%d] (%s) function[%d]: file is required", i, app.Name, j, feature.Name, k))
				}
				if fn.Name == "" {
					errors = append(errors, fmt.Sprintf("app[%d] (%s) feature[%d] (%s) function[%d]: name is required", i, app.Name, j, feature.Name, k))
				}
			}
		}
	}

	for i, mod := range a.Modules {
		if mod.Path == "" {
			errors = append(errors, fmt.Sprintf("module[%d]: path is required", i))
		}
		if mod.Description == "" {
			errors = append(errors, fmt.Sprintf("module[%d] (%s): description is required", i, mod.Path))
		}
		if mod.Type == "" {
			errors = append(errors, fmt.Sprintf("module[%d] (%s): type is required", i, mod.Path))
		} else if mod.Type != "domain" && mod.Type != "implementation" {
			errors = append(errors, fmt.Sprintf("module[%d] (%s): type must be 'domain' or 'implementation', got '%s'", i, mod.Path, mod.Type))
		}
	}

	return errors
}

func Load(path string) (*Architecture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read architecture file %s: %w", path, err)
	}

	var arch Architecture
	if err := yaml.Unmarshal(data, &arch); err != nil {
		return nil, fmt.Errorf("failed to parse architecture YAML: %w", err)
	}

	return &arch, nil
}
