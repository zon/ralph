package review

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandLoop(t *testing.T) {
	t.Run("unknown loop type returns error", func(t *testing.T) {
		_, err := ExpandLoop("unknown-type", "/some/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown loop type")
	})
}

func TestExpandDomainFunctionLoop(t *testing.T) {
	t.Run("empty architecture returns empty iterations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		require.NoError(t, os.WriteFile(path, []byte("apps: []\nmodules: []\n"), 0644))

		iterations, err := expandDomainFunctionLoop(path)
		require.NoError(t, err)
		assert.Empty(t, iterations)
	})

	t.Run("deduplicates same name and file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		archContent := `
apps:
  - name: test-app
    description: Test app
    main:
      file: cmd/main.go
      function: main
    features:
      - name: Feature1
        description: Feature 1
        functions:
          - file: internal/pkg/pkg.go
            name: DoThing
      - name: Feature2
        description: Feature 2
        functions:
          - file: internal/pkg/pkg.go
            name: DoThing
`
		require.NoError(t, os.WriteFile(path, []byte(archContent), 0644))

		iterations, err := expandDomainFunctionLoop(path)
		require.NoError(t, err)
		require.Len(t, iterations, 1)
		assert.Equal(t, "DoThing", iterations[0].FunctionName)
		assert.Equal(t, "internal/pkg/pkg.go", iterations[0].FunctionPath)
	})

	t.Run("same name with different file creates separate entries", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		archContent := `
apps:
  - name: test-app
    description: Test app
    main:
      file: cmd/main.go
      function: main
    features:
      - name: Feature1
        description: Feature 1
        functions:
          - file: internal/pkg/pkg.go
            name: DoThing
      - name: Feature2
        description: Feature 2
        functions:
          - file: internal/other/other.go
            name: DoThing
`
		require.NoError(t, os.WriteFile(path, []byte(archContent), 0644))

		iterations, err := expandDomainFunctionLoop(path)
		require.NoError(t, err)
		require.Len(t, iterations, 2)
		assert.Equal(t, "DoThing", iterations[0].FunctionName)
		assert.Equal(t, "internal/pkg/pkg.go", iterations[0].FunctionPath)
		assert.Equal(t, "DoThing", iterations[1].FunctionName)
		assert.Equal(t, "internal/other/other.go", iterations[1].FunctionPath)
	})

	t.Run("returns ordered iterations across apps and features", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		archContent := `
apps:
  - name: app1
    description: App 1
    main:
      file: cmd/main.go
      function: main
    features:
      - name: Feature1
        description: Feature 1
        functions:
          - file: internal/pkg/first.go
            name: FirstFunc
  - name: app2
    description: App 2
    main:
      file: cmd/main.go
      function: main
    features:
      - name: Feature2
        description: Feature 2
        functions:
          - file: internal/pkg/second.go
            name: SecondFunc
`
		require.NoError(t, os.WriteFile(path, []byte(archContent), 0644))

		iterations, err := expandDomainFunctionLoop(path)
		require.NoError(t, err)
		require.Len(t, iterations, 2)
		assert.Equal(t, "FirstFunc", iterations[0].FunctionName)
		assert.Equal(t, "internal/pkg/first.go", iterations[0].FunctionPath)
		assert.Equal(t, "SecondFunc", iterations[1].FunctionName)
		assert.Equal(t, "internal/pkg/second.go", iterations[1].FunctionPath)
	})
}
