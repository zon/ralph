package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// loadConfigWithContent writes a .ralph/config.yaml with the given content and
// loads it with LoadConfig.
func loadConfigWithContent(t *testing.T, content string) *RalphConfig {
	t.Helper()
	cfg, err := loadConfigErrorWithContent(t, content)
	require.NoError(t, err)
	return cfg
}

// loadConfigErrorWithContent writes a .ralph/config.yaml with the given content
// and returns the error LoadConfig reports for it, if any.
func loadConfigErrorWithContent(t *testing.T, content string) (*RalphConfig, error) {
	t.Helper()
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(content), 0644))
	t.Chdir(tmpDir)

	return LoadConfig()
}

func TestResolveItems_ConfigQueryUsedWhenNoFlag(t *testing.T) {
	// GIVEN `items: .requirements` is set in `.ralph/config.yaml`
	cfg := loadConfigWithContent(t, "items: .requirements\n")

	// AND no `--items` flag is passed
	// WHEN the item query is resolved
	resolved := cfg.ResolveItems("")

	// THEN the resolved query is `.requirements`
	assert.Equal(t, ".requirements", resolved)
}

func TestResolveItems_FlagOverridesConfig(t *testing.T) {
	cfg := loadConfigWithContent(t, "items: .requirements\n")
	assert.Equal(t, ".spec.tasks", cfg.ResolveItems(".spec.tasks"))
}

func TestResolveItems_DefaultsToDotWhenFlagAndConfigUnset(t *testing.T) {
	cfg := loadConfigWithContent(t, "")
	assert.Equal(t, ".", cfg.ResolveItems(""))
}

func TestLoadConfig_ItemsFieldParsed(t *testing.T) {
	cfg := loadConfigWithContent(t, "items: .spec.tasks\n")
	assert.Equal(t, ".spec.tasks", cfg.Items)
}

func TestLoadConfig_ItemsDefaultsToDot(t *testing.T) {
	cfg := loadConfigWithContent(t, "")
	assert.Equal(t, ".", cfg.Items)
}

func TestLoadConfig_CleanupFieldParsed(t *testing.T) {
	cfg := loadConfigWithContent(t, "cleanup: true\n")
	assert.True(t, cfg.Cleanup)
}

func TestLoadConfig_CleanupDefaultsToTrue(t *testing.T) {
	cfg := loadConfigWithContent(t, "")
	assert.True(t, cfg.Cleanup, "cleanup should be enabled when the config file does not set it")
}

func TestLoadConfig_CleanupDisabledByConfig(t *testing.T) {
	cfg := loadConfigWithContent(t, "cleanup: false\n")
	assert.False(t, cfg.Cleanup, "cleanup: false in the config file disables project file cleanup")
}

func TestLoadConfig_CleanupDefaultsToTrueWhenNoConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.True(t, cfg.Cleanup, "cleanup should default to enabled when no .ralph directory exists")
}

func TestConfigItemsSerializedWhenSet(t *testing.T) {
	cfg := &RalphConfig{Items: ".requirements"}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(out), "items: .requirements")
}

func TestConfigItemsOmittedWhenEmpty(t *testing.T) {
	cfg := &RalphConfig{}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "items:")
}

func TestConfigCleanupSerializedWhenSet(t *testing.T) {
	cfg := &RalphConfig{Cleanup: true}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(out), "cleanup: true")
}

func TestConfigCleanupOmittedWhenFalse(t *testing.T) {
	cfg := &RalphConfig{}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "cleanup")
}
