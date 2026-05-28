package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAny_ReturnsDefaultConfig(t *testing.T) {
	cfg := Any()

	assert.Equal(t, 10, cfg.MaxIterations)
	assert.Equal(t, "main", cfg.DefaultBranch)
	assert.Equal(t, "deepseek/deepseek-chat", cfg.Model)
	assert.Equal(t, "ralph-zon", cfg.App.Name)
	assert.Equal(t, "2966665", cfg.App.ID)
	assert.Empty(t, cfg.Before)
	assert.Empty(t, cfg.Services)
	assert.NotEmpty(t, cfg.Instructions)
	assert.True(t, strings.Contains(cfg.Instructions, "## Instructions"))
	assert.NotEmpty(t, cfg.CommentInstructions)
	assert.True(t, strings.Contains(cfg.CommentInstructions, "# Comment Instructions"))
	assert.NotEmpty(t, cfg.MergeInstructions)
	assert.True(t, strings.Contains(cfg.MergeInstructions, "# Merge Instructions"))
}

func TestAny_ReturnsValidNonNilConfig(t *testing.T) {
	cfg := Any()
	assert.NotNil(t, cfg)
}
