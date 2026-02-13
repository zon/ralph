package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
)

func TestRunAgentDryRun(t *testing.T) {
	ctx := context.NewContext(true, false, true, false)

	err := RunAgent(ctx, "test prompt")
	if err != nil {
		t.Errorf("RunAgent in dry-run mode should not fail: %v", err)
	}
}

func TestRunAgentNoSecrets(t *testing.T) {
	// Create a temp directory and change to it
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create .ralph directory
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.Mkdir(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph dir: %v", err)
	}

	// Create minimal config
	configContent := `llmProvider: deepseek`
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	ctx := context.NewContext(false, false, true, false)

	err = RunAgent(ctx, "test prompt")
	if err == nil {
		t.Error("RunAgent should fail without secrets")
	}
}

func TestGeneratePRSummaryDryRun(t *testing.T) {
	ctx := context.NewContext(true, false, true, false)

	summary, err := GeneratePRSummary(ctx, "test.yaml", 3)
	if err != nil {
		t.Errorf("GeneratePRSummary in dry-run mode should not fail: %v", err)
	}

	if summary != "dry-run-pr-summary" {
		t.Errorf("Expected dry-run-pr-summary, got: %s", summary)
	}
}

func TestGeneratePRSummaryNoProject(t *testing.T) {
	ctx := context.NewContext(false, false, true, false)

	_, err := GeneratePRSummary(ctx, "nonexistent.yaml", 1)
	if err == nil {
		t.Error("GeneratePRSummary should fail with nonexistent project file")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.RalphConfig
		secrets   *config.RalphSecrets
		shouldErr bool
	}{
		{
			name: "valid deepseek config",
			config: &config.RalphConfig{
				LLMProvider: "deepseek",
			},
			secrets: &config.RalphSecrets{
				APIKeys: map[string]string{
					"deepseek": "sk-test",
				},
			},
			shouldErr: false,
		},
		{
			name: "missing api key",
			config: &config.RalphConfig{
				LLMProvider: "deepseek",
			},
			secrets: &config.RalphSecrets{
				APIKeys: map[string]string{},
			},
			shouldErr: true,
		},
		{
			name:   "default provider with valid key",
			config: &config.RalphConfig{
				// No provider specified, defaults to deepseek
			},
			secrets: &config.RalphSecrets{
				APIKeys: map[string]string{
					"deepseek": "sk-test",
				},
			},
			shouldErr: false,
		},
		{
			name: "empty api key",
			config: &config.RalphConfig{
				LLMProvider: "deepseek",
			},
			secrets: &config.RalphSecrets{
				APIKeys: map[string]string{
					"deepseek": "",
				},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config, tt.secrets)
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
