package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigOptionalAdapterLoadOptional(t *testing.T) {
	t.Run("returns an error for an invalid config", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("mode: [\n"), 0o644))
		t.Chdir(dir)

		cfg, err := (&configOptionalAdapter{}).LoadOptional()

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to load config")
	})

	t.Run("returns the default config when there is no config file", func(t *testing.T) {
		t.Chdir(t.TempDir())

		cfg, err := (&configOptionalAdapter{}).LoadOptional()

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.True(t, cfg.Cleanup)
		assert.NotEmpty(t, cfg.Mode)
	})
}
