package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/k8s"
	"gopkg.in/yaml.v3"
)

// TestLoadConfig_OpenCodeSecretDefaults asserts workflow.opencodeSecret
// resolves to the Secret ralph setup writes when the config omits it.
func TestLoadConfig_OpenCodeSecretDefaults(t *testing.T) {
	cfg := loadConfigWithContent(t, "")

	assert.Equal(t, k8s.OpenCodeSecretName, cfg.Workflow.OpenCodeSecret)
}

// TestLoadConfig_OpenCodeSecretParsed asserts a configured Secret name is kept.
func TestLoadConfig_OpenCodeSecretParsed(t *testing.T) {
	cfg := loadConfigWithContent(t, "workflow:\n  opencodeSecret: opencode-ai-gateway\n")

	assert.Equal(t, "opencode-ai-gateway", cfg.Workflow.OpenCodeSecret)
}

// TestLoadConfig_OpenCodeSecretEmptyRejected asserts an explicitly empty Secret
// name fails validation with an error naming the field.
func TestLoadConfig_OpenCodeSecretEmptyRejected(t *testing.T) {
	_, err := loadConfigErrorWithContent(t, "workflow:\n  opencodeSecret: \"\"\n")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencodeSecret")
}

// TestOpenCodeSecretYAMLRoundTrip asserts a configured Secret survives a
// marshal and unmarshal cycle.
func TestOpenCodeSecretYAMLRoundTrip(t *testing.T) {
	cfg := &RalphConfig{Workflow: WorkflowConfig{OpenCodeSecret: "opencode-ai-gateway"}}

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "opencodeSecret: opencode-ai-gateway")

	var decoded RalphConfig
	require.NoError(t, yaml.Unmarshal(data, &decoded))
	assert.Equal(t, "opencode-ai-gateway", decoded.Workflow.OpenCodeSecret)
}
