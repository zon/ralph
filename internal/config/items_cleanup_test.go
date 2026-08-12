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
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(content), 0644))
	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	return cfg
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

func TestResolveCleanup_ConfigValueUsedWhenNoFlag(t *testing.T) {
	// GIVEN `cleanup: true` is set in `.ralph/config.yaml`
	cfg := loadConfigWithContent(t, "cleanup: true\n")

	// AND no `--cleanup` flag is passed
	// WHEN cleanup is resolved
	resolved := cfg.ResolveCleanup(nil)

	// THEN cleanup is enabled
	assert.True(t, resolved)
}

func TestResolveItems_FlagOverridesConfig(t *testing.T) {
	cfg := loadConfigWithContent(t, "items: .requirements\n")
	assert.Equal(t, ".spec.tasks", cfg.ResolveItems(".spec.tasks"))
}

func TestResolveItems_DefaultsToDotWhenFlagAndConfigUnset(t *testing.T) {
	cfg := loadConfigWithContent(t, "")
	assert.Equal(t, ".", cfg.ResolveItems(""))
}

func TestResolveCleanup_FlagOverridesConfig(t *testing.T) {
	cfg := loadConfigWithContent(t, "cleanup: true\n")
	flag := false
	assert.False(t, cfg.ResolveCleanup(&flag))
}

func TestResolveCleanup_DisabledWhenFlagAndConfigUnset(t *testing.T) {
	cfg := loadConfigWithContent(t, "")
	assert.False(t, cfg.ResolveCleanup(nil))
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

func TestLoadConfig_CleanupDefaultsToFalse(t *testing.T) {
	cfg := loadConfigWithContent(t, "")
	assert.False(t, cfg.Cleanup)
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

func TestDefaultInstructionsExplainCompletionTrailer(t *testing.T) {
	instructions := DefaultDevelopmentInstructions()
	assert.Contains(t, instructions, "completion trailer")
	assert.Contains(t, instructions, "only way")
}
