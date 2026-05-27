package architecture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSpecFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func TestLoadSpec(t *testing.T) {
	t.Run("loads valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		writeSpecFile(t, path, `
modules:
  - path: internal/foo
    description: Foo module
    orchestration: true
  - path: internal/bar
    description: Bar module
`)
		arch, err := LoadSpec(path)
		require.NoError(t, err)
		require.NotNil(t, arch)
		require.Len(t, arch.Modules, 2)
		assert.Equal(t, "internal/foo", arch.Modules[0].Path)
		assert.Equal(t, "Foo module", arch.Modules[0].Description)
		assert.True(t, arch.Modules[0].Orchestration)
		assert.Equal(t, "internal/bar", arch.Modules[1].Path)
		assert.False(t, arch.Modules[1].Orchestration)
	})

	t.Run("returns nil for missing file", func(t *testing.T) {
		arch, err := LoadSpec("/nonexistent/architecture.yaml")
		require.NoError(t, err)
		assert.Nil(t, arch)
	})

	t.Run("error on invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		writeSpecFile(t, path, "modules: [invalid\n")
		_, err := LoadSpec(path)
		require.Error(t, err)
	})
}

func TestSaveSpec(t *testing.T) {
	t.Run("writes readable file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "architecture.yaml")
		arch := &SpecArchitecture{
			Modules: []SpecModule{
				{Path: "internal/foo", Description: "Foo module", Orchestration: true},
				{Path: "internal/bar", Description: "Bar module"},
			},
		}
		require.NoError(t, SaveSpec(path, arch))

		loaded, err := LoadSpec(path)
		require.NoError(t, err)
		require.Len(t, loaded.Modules, 2)
		assert.Equal(t, "internal/foo", loaded.Modules[0].Path)
		assert.True(t, loaded.Modules[0].Orchestration)
		assert.Equal(t, "internal/bar", loaded.Modules[1].Path)
		assert.False(t, loaded.Modules[1].Orchestration)
	})
}

func TestMigrateImplementedModules(t *testing.T) {
	t.Run("no-op when feature architecture missing", func(t *testing.T) {
		dir := t.TempDir()
		count, err := MigrateImplementedModules(
			filepath.Join(dir, "feature/architecture.yaml"),
			filepath.Join(dir, "specs/architecture.yaml"),
			dir,
		)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("skips module whose path does not exist", func(t *testing.T) {
		dir := t.TempDir()
		featurePath := filepath.Join(dir, "feature/architecture.yaml")
		globalPath := filepath.Join(dir, "specs/architecture.yaml")

		writeSpecFile(t, featurePath, `
modules:
  - path: internal/notyet
    description: Not yet implemented
`)
		writeSpecFile(t, globalPath, `
modules:
  - path: internal/existing
    description: Existing module
`)
		count, err := MigrateImplementedModules(featurePath, globalPath, dir)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Feature architecture unchanged
		feature, _ := LoadSpec(featurePath)
		require.Len(t, feature.Modules, 1)
	})

	t.Run("migrates new module when directory exists", func(t *testing.T) {
		dir := t.TempDir()
		featurePath := filepath.Join(dir, "feature/architecture.yaml")
		globalPath := filepath.Join(dir, "specs/architecture.yaml")

		writeSpecFile(t, featurePath, `
modules:
  - path: internal/newmod
    description: New implementation module
`)
		writeSpecFile(t, globalPath, `
modules:
  - path: internal/existing
    description: Existing module
`)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal/newmod"), 0755))

		count, err := MigrateImplementedModules(featurePath, globalPath, dir)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		global, _ := LoadSpec(globalPath)
		require.Len(t, global.Modules, 2)
		assert.Equal(t, "internal/existing", global.Modules[0].Path)
		assert.Equal(t, "internal/newmod", global.Modules[1].Path)

		// Feature architecture removed (all entries migrated)
		_, statErr := os.Stat(featurePath)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("replaces existing global entry when paths match", func(t *testing.T) {
		dir := t.TempDir()
		featurePath := filepath.Join(dir, "feature/architecture.yaml")
		globalPath := filepath.Join(dir, "specs/architecture.yaml")

		writeSpecFile(t, featurePath, `
modules:
  - path: internal/git
    description: Updated description from feature.
`)
		writeSpecFile(t, globalPath, `
modules:
  - path: internal/git
    description: Old description.
`)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal/git"), 0755))

		count, err := MigrateImplementedModules(featurePath, globalPath, dir)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		global, _ := LoadSpec(globalPath)
		require.Len(t, global.Modules, 1)
		assert.Equal(t, "internal/git", global.Modules[0].Path)
		assert.Equal(t, "Updated description from feature.", global.Modules[0].Description)
	})

	t.Run("partial migration leaves remaining in feature", func(t *testing.T) {
		dir := t.TempDir()
		featurePath := filepath.Join(dir, "feature/architecture.yaml")
		globalPath := filepath.Join(dir, "specs/architecture.yaml")

		writeSpecFile(t, featurePath, `
modules:
  - path: internal/done
    description: Done module.
  - path: internal/notyet
    description: Not yet module.
`)
		writeSpecFile(t, globalPath, `
modules:
  - path: internal/existing
    description: Existing.
`)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal/done"), 0755))

		count, err := MigrateImplementedModules(featurePath, globalPath, dir)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		global, _ := LoadSpec(globalPath)
		require.Len(t, global.Modules, 2)
		assert.Equal(t, "internal/done", global.Modules[1].Path)

		feature, _ := LoadSpec(featurePath)
		require.Len(t, feature.Modules, 1)
		assert.Equal(t, "internal/notyet", feature.Modules[0].Path)
	})

	t.Run("no-op when feature has no modules", func(t *testing.T) {
		dir := t.TempDir()
		featurePath := filepath.Join(dir, "feature/architecture.yaml")
		globalPath := filepath.Join(dir, "specs/architecture.yaml")

		writeSpecFile(t, featurePath, `modules: []`)
		writeSpecFile(t, globalPath, `
modules:
  - path: internal/existing
    description: Existing.
`)

		count, err := MigrateImplementedModules(featurePath, globalPath, dir)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
