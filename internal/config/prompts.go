package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Prompt is either inline prompt text or a file path relative to the
// repository root.
type Prompt struct {
	Text string
	File string
}

// UnmarshalYAML implements yaml.Unmarshaler so a prompt accepts either a
// string or a mapping with a file.
func (p *Prompt) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var text string
		if err := node.Decode(&text); err != nil {
			return err
		}
		p.Text = text
		p.File = ""
		return nil
	case yaml.MappingNode:
		var value struct {
			File string `yaml:"file"`
		}
		if err := node.Decode(&value); err != nil {
			return err
		}
		p.Text = ""
		p.File = value.File
		return nil
	}
	return fmt.Errorf("line %d: prompt must be a string or a mapping with a file", node.Line)
}

// promptKey normalizes an event name or prompt key for comparison by
// lowercasing it and removing underscores and hyphens.
func promptKey(value string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(value))
}

// PromptForEvent returns the prompt for an event. It reads a file prompt
// relative to the repository root and fails when the event has no key, two keys
// normalize alike, or the prompt is empty.
func (c *RalphConfig) PromptForEvent(event string) (string, error) {
	key, err := c.promptKeyForEvent(event)
	if err != nil {
		return "", err
	}

	prompt := c.Prompts[key]
	if prompt.File == "" {
		if strings.TrimSpace(prompt.Text) == "" {
			return "", fmt.Errorf("prompt for event %q is empty", event)
		}
		return prompt.Text, nil
	}

	root, err := c.promptRoot()
	if err != nil {
		return "", fmt.Errorf("prompt for event %q: %w", event, err)
	}
	data, err := os.ReadFile(filepath.Join(root, prompt.File))
	if err != nil {
		return "", fmt.Errorf("prompt for event %q names file %q: %w", event, prompt.File, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("prompt for event %q names empty file %q", event, prompt.File)
	}
	return string(data), nil
}

// promptKeyForEvent returns the prompt key matching the event. It fails when
// the event has no key or two keys normalize alike.
func (c *RalphConfig) promptKeyForEvent(event string) (string, error) {
	normalized := promptKey(event)
	keys := make([]string, 0, len(c.Prompts))
	for key := range c.Prompts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	seen := make(map[string]string, len(keys))
	for _, key := range keys {
		form := promptKey(key)
		if previous, ok := seen[form]; ok {
			return "", fmt.Errorf("prompts keys %q and %q normalize alike", previous, key)
		}
		seen[form] = key
	}

	key, ok := seen[normalized]
	if !ok {
		return "", fmt.Errorf("no prompt for event %q", event)
	}
	return key, nil
}

// promptRoot returns the repository root holding the config file. Without a
// loaded config file, the current working directory is the root.
func (c *RalphConfig) promptRoot() (string, error) {
	if c.ConfigPath == "" {
		return os.Getwd()
	}
	return filepath.Dir(filepath.Dir(c.ConfigPath)), nil
}
