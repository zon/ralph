package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig_WithPrompts asserts the top-level prompts map accepts both
// inline text and a file reference.
func TestLoadConfig_WithPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `prompts:
  push: |
    Review the repository and make one small improvement.
  pullRequest:
    file: .ralph/prompts/pull-request.md
  schedule:
    file: .ralph/prompts/daily.md
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	require.Len(t, cfg.Prompts, 3)
	assert.Equal(t, "Review the repository and make one small improvement.\n", cfg.Prompts["push"].Text)
	assert.Empty(t, cfg.Prompts["push"].File)
	assert.Equal(t, ".ralph/prompts/pull-request.md", cfg.Prompts["pullRequest"].File)
	assert.Equal(t, ".ralph/prompts/daily.md", cfg.Prompts["schedule"].File)
}

// TestPromptForEventNormalizesKeys asserts the event lookup lowercases the
// event and each key and removes underscores and hyphens before comparing.
func TestPromptForEventNormalizesKeys(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		event string
	}{
		{name: "same case", key: "pullRequest", event: "pullRequest"},
		{name: "github event", key: "pullRequest", event: "pull_request"},
		{name: "hyphenated event", key: "pullRequest", event: "pull-request"},
		{name: "mixed case event", key: "pullRequest", event: "Pull_Request"},
		{name: "no separator event", key: "pullRequest", event: "PULLREQUEST"},
		{name: "mixed case key", key: "Pull_Request", event: "pull_request"},
		{name: "schedule key", key: "schedule", event: "schedule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RalphConfig{
				Prompts: map[string]Prompt{
					tt.key: {Text: "do work"},
				},
			}

			prompt, err := cfg.PromptForEvent(tt.event)
			require.NoError(t, err)
			assert.Equal(t, "do work", prompt)
		})
	}
}

// TestPromptForEventDuplicateNormalizedKeys asserts a lookup fails with an
// error naming both keys when two keys normalize alike.
func TestPromptForEventDuplicateNormalizedKeys(t *testing.T) {
	cfg := &RalphConfig{
		Prompts: map[string]Prompt{
			"pullRequest":  {Text: "one"},
			"pull_request": {Text: "two"},
		},
	}

	_, err := cfg.PromptForEvent("pull_request")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pullRequest")
	assert.Contains(t, err.Error(), "pull_request")
}

// TestPromptForEventErrorsNameEvent asserts a missing key, an empty prompt, and
// a missing or empty file each fail with an error that names the event.
func TestPromptForEventErrorsNameEvent(t *testing.T) {
	tests := []struct {
		name   string
		event  string
		prompt Prompt
		file   string
	}{
		{name: "missing key", event: "issue_comment"},
		{name: "empty inline prompt", event: "push", prompt: Prompt{Text: "  \n\t"}},
		{name: "empty file", event: "schedule", prompt: Prompt{File: "empty.md"}, file: "  \n"},
		{name: "missing file", event: "pull_request", prompt: Prompt{File: "missing.md"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(root, ".ralph"), 0755))
			if tt.file != "" {
				require.NoError(t, os.WriteFile(filepath.Join(root, tt.prompt.File), []byte(tt.file), 0644))
			}

			cfg := &RalphConfig{
				ConfigPath: filepath.Join(root, ".ralph", "config.yaml"),
				Prompts:    map[string]Prompt{tt.event: tt.prompt},
			}

			_, err := cfg.PromptForEvent(tt.event)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.event)
		})
	}
}

// TestPromptForEventReadsFileRelativeToRepositoryRoot asserts a file prompt is
// read from the repository root even when the process runs elsewhere.
func TestPromptForEventReadsFileRelativeToRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".ralph", "prompts"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".ralph", "prompts", "daily.md"), []byte("run the daily check"), 0644))

	cfg := &RalphConfig{
		ConfigPath: filepath.Join(root, ".ralph", "config.yaml"),
		Prompts: map[string]Prompt{
			"schedule": {File: ".ralph/prompts/daily.md"},
		},
	}

	t.Chdir(t.TempDir())

	prompt, err := cfg.PromptForEvent("schedule")

	require.NoError(t, err)
	assert.Equal(t, "run the daily check", prompt)
}

// TestLoadConfig_PromptsDoNotFailOtherCommands asserts loading the config does
// not resolve or validate prompts, so commands other than the launcher ignore
// the section.
func TestLoadConfig_PromptsDoNotFailOtherCommands(t *testing.T) {
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `prompts:
  pullRequest: one
  pull_request: two
  schedule:
    file: missing.md
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := LoadConfig()

	require.NoError(t, err)
	require.Len(t, cfg.Prompts, 3)
}
