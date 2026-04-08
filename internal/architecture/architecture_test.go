package architecture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validArchitecture = `
apps:
  - name: ralph
    description: AI-powered development agent
    main:
      file: cmd/ralph/main.go
      function: main
    features:
      - name: Project Management
        description: Manages development projects with requirements tracking
        functions:
          - file: internal/project/project.go
            name: LoadProject
          - file: internal/project/project.go
            name: SaveProject
      - name: Code Review
        description: AI-powered code review from config prompts
        functions:
          - file: internal/ai/ai.go
            name: RunAgent
modules:
  - path: internal/domain
    description: Core business logic and models
    type: domain
  - path: internal/ai
    description: AI agent integration via OpenCode
    type: implementation
`

func TestLoad(t *testing.T) {
	t.Run("loads valid architecture", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		require.NoError(t, os.WriteFile(path, []byte(validArchitecture), 0644))

		arch, err := Load(path)
		require.NoError(t, err)

		require.Len(t, arch.Apps, 1)
		assert.Equal(t, "ralph", arch.Apps[0].Name)
		assert.Equal(t, "AI-powered development agent", arch.Apps[0].Description)
		assert.Equal(t, "cmd/ralph/main.go", arch.Apps[0].Main.File)
		assert.Equal(t, "main", arch.Apps[0].Main.Function)
		require.Len(t, arch.Apps[0].Features, 2)
		assert.Equal(t, "Project Management", arch.Apps[0].Features[0].Name)
		require.Len(t, arch.Apps[0].Features[0].Functions, 2)
		assert.Equal(t, "internal/project/project.go", arch.Apps[0].Features[0].Functions[0].File)
		assert.Equal(t, "LoadProject", arch.Apps[0].Features[0].Functions[0].Name)

		require.Len(t, arch.Modules, 2)
		assert.Equal(t, "internal/domain", arch.Modules[0].Path)
		assert.Equal(t, "Core business logic and models", arch.Modules[0].Description)
		assert.Equal(t, "domain", arch.Modules[0].Type)
	})

	t.Run("error on missing file", func(t *testing.T) {
		_, err := Load("/nonexistent/architecture.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read architecture file")
	})

	t.Run("error on invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		require.NoError(t, os.WriteFile(path, []byte("apps: [invalid yaml\n"), 0644))

		_, err := Load(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse architecture YAML")
	})
}

func TestArchitecture_Validate(t *testing.T) {
	t.Run("valid architecture returns no errors", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		require.NoError(t, os.WriteFile(path, []byte(validArchitecture), 0644))

		arch, err := Load(path)
		require.NoError(t, err)

		errors := arch.Validate()
		assert.Empty(t, errors)
	})

	t.Run("app name is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features:    []Feature{{Name: "feat", Description: "desc", Functions: []FuncRef{{File: "f.go", Name: "fn"}}}},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "name is required")
	})

	t.Run("app description is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features:    []Feature{{Name: "feat", Description: "desc", Functions: []FuncRef{{File: "f.go", Name: "fn"}}}},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "description is required")
	})

	t.Run("app main file is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "", Function: "main"},
					Features:    []Feature{{Name: "feat", Description: "desc", Functions: []FuncRef{{File: "f.go", Name: "fn"}}}},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "main.file is required")
	})

	t.Run("app main function is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: ""},
					Features:    []Feature{{Name: "feat", Description: "desc", Functions: []FuncRef{{File: "f.go", Name: "fn"}}}},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "main.function is required")
	})

	t.Run("app must have at least one feature", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features:    []Feature{},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "at least one feature is required")
	})

	t.Run("feature name is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features: []Feature{
						{
							Name:        "",
							Description: "feature description",
							Functions:   []FuncRef{{File: "f.go", Name: "fn"}},
						},
					},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "name is required")
	})

	t.Run("feature description is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features: []Feature{
						{
							Name:        "feature-name",
							Description: "",
							Functions:   []FuncRef{{File: "f.go", Name: "fn"}},
						},
					},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "description is required")
	})

	t.Run("feature must have at least one function ref", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features: []Feature{
						{
							Name:        "feature-name",
							Description: "feature description",
							Functions:   []FuncRef{},
						},
					},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "at least one function ref is required")
	})

	t.Run("function ref file is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features: []Feature{
						{
							Name:        "feature-name",
							Description: "feature description",
							Functions: []FuncRef{
								{File: "", Name: "SomeFunc"},
							},
						},
					},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "file is required")
	})

	t.Run("function ref name is required", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "test-app",
					Description: "test description",
					Main:        MainFunc{File: "cmd/main.go", Function: "main"},
					Features: []Feature{
						{
							Name:        "feature-name",
							Description: "feature description",
							Functions: []FuncRef{
								{File: "internal/pkg/pkg.go", Name: ""},
							},
						},
					},
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "name is required")
	})

	t.Run("module path is required", func(t *testing.T) {
		arch := &Architecture{
			Modules: []Module{
				{
					Path:        "",
					Description: "module description",
					Type:        "domain",
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "path is required")
	})

	t.Run("module description is required", func(t *testing.T) {
		arch := &Architecture{
			Modules: []Module{
				{
					Path:        "internal/pkg",
					Description: "",
					Type:        "domain",
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "description is required")
	})

	t.Run("module type is required", func(t *testing.T) {
		arch := &Architecture{
			Modules: []Module{
				{
					Path:        "internal/pkg",
					Description: "module description",
					Type:        "",
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "type is required")
	})

	t.Run("module type must be domain or implementation", func(t *testing.T) {
		arch := &Architecture{
			Modules: []Module{
				{
					Path:        "internal/pkg",
					Description: "module description",
					Type:        "invalid",
				},
			},
		}

		errors := arch.Validate()
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0], "type must be 'domain' or 'implementation'")
	})

	t.Run("multiple errors collected", func(t *testing.T) {
		arch := &Architecture{
			Apps: []App{
				{
					Name:        "",
					Description: "",
					Main:        MainFunc{File: "", Function: ""},
					Features:    []Feature{},
				},
			},
			Modules: []Module{
				{
					Path:        "",
					Description: "",
					Type:        "",
				},
			},
		}

		errors := arch.Validate()
		assert.Len(t, errors, 8)
	})
}
