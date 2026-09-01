package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestAny_ReturnsDefaultConfig(t *testing.T) {
	cfg := Any()

	assert.Equal(t, "main", cfg.DefaultBranch)
	assert.Equal(t, "deepseek/deepseek-chat", cfg.Model)
	assert.Equal(t, "ralph-zon", cfg.App.Name)
	assert.Equal(t, "2966665", cfg.App.ID)
	assert.Empty(t, cfg.Before)
	assert.Empty(t, cfg.Services)
	assert.Empty(t, cfg.Instructions, "development instructions come from the prompt unless the repository supplies its own")
}

func TestAny_ReturnsValidNonNilConfig(t *testing.T) {
	cfg := Any()
	assert.NotNil(t, cfg)
}

func TestWithVariant_SetsVariantField(t *testing.T) {
	cfg := WithVariant("high")
	assert.Equal(t, "high", cfg.Variant)
}

func TestWithVariant_ReturnsBaselineConfig(t *testing.T) {
	cfg := WithVariant("custom")
	assert.Equal(t, "main", cfg.DefaultBranch)
	assert.Equal(t, "deepseek/deepseek-chat", cfg.Model)
}

func TestWithVariant_EmptyString(t *testing.T) {
	cfg := WithVariant("")
	assert.Empty(t, cfg.Variant)
}

func TestRalphConfig_VariantYAMLRoundTrip(t *testing.T) {
	cfg := WithVariant("high")
	data, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "variant: high")

	var decoded RalphConfig
	err = yaml.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "high", decoded.Variant)
}
