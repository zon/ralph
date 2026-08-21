package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()

	t.Chdir(tmpDir)

	// Create .ralph directory to satisfy new LoadConfig requirement
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".ralph"), 0755))

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "main", config.DefaultBranch)
	assert.Empty(t, config.Services)
	assert.Empty(t, config.Instructions, "LoadConfig() Instructions must stay unset so the prompt supplies its default steps")
	assert.True(t, strings.Contains(config.CommentInstructions, "# Comment Instructions"), "LoadConfig() CommentInstructions missing expected header")
}

func TestLoadConfig_FromFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Chdir(tmpDir)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	// Write config file
	configContent := `maxIterations: 5
defaultBranch: develop
services:
  - name: test-service
    command: echo
    args: [hello]
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Write custom instructions file
	instructionsContent := "Custom instructions for testing"
	instructionsPath := filepath.Join(ralphDir, "instructions.md")
	require.NoError(t, os.WriteFile(instructionsPath, []byte(instructionsContent), 0644))

	// Change to temp directory
	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "develop", config.DefaultBranch)
	assert.Len(t, config.Services, 1)
	assert.Equal(t, "test-service", config.Services[0].Name)
	assert.Equal(t, instructionsContent, config.Instructions)
}

func TestLoadConfig_WithWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `before:
  - name: compile
    command: make
    args: [build]
    workDir: /tmp/myapp
services:
  - name: api
    command: ./api
    workDir: /tmp/myapp/bin
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, cfg.Before, 1)
	assert.Equal(t, "/tmp/myapp", cfg.Before[0].WorkDir)

	require.Len(t, cfg.Services, 1)
	assert.Equal(t, "/tmp/myapp/bin", cfg.Services[0].WorkDir)
}

func TestLoadConfig_WorkDirOmitted(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `before:
  - name: compile
    command: make
services:
  - name: api
    command: ./api
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, cfg.Before, 1)
	assert.Equal(t, "", cfg.Before[0].WorkDir)

	require.Len(t, cfg.Services, 1)
	assert.Equal(t, "", cfg.Services[0].WorkDir)
}

func TestLoadConfig_WithWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 5
defaultBranch: main
workflow:
  image:
    repository: ghcr.io/example/ralph-runner
    tag: v1.0.0
  configMaps:
    - name: app-config
    - name: shared-data
  secrets:
    - name: api-keys
    - name: database-creds
  env:
    LOG_LEVEL: debug
    APP_ENV: production
  context: my-k8s-cluster
  namespace: ralph-workflows
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	// Verify workflow image config
	assert.Equal(t, "ghcr.io/example/ralph-runner", config.Workflow.Image.Repository)
	assert.Equal(t, "v1.0.0", config.Workflow.Image.Tag)

	// Verify configMaps
	require.Len(t, config.Workflow.ConfigMaps, 2)
	assert.Equal(t, "app-config", config.Workflow.ConfigMaps[0].Name)
	assert.Equal(t, "shared-data", config.Workflow.ConfigMaps[1].Name)

	// Verify secrets
	require.Len(t, config.Workflow.Secrets, 2)
	assert.Equal(t, "api-keys", config.Workflow.Secrets[0].Name)
	assert.Equal(t, "database-creds", config.Workflow.Secrets[1].Name)

	// Verify environment variables
	require.Len(t, config.Workflow.Env, 2)
	assert.Equal(t, "debug", config.Workflow.Env["LOG_LEVEL"].Value)
	assert.Equal(t, "production", config.Workflow.Env["APP_ENV"].Value)

	// Verify Kubernetes context and namespace
	assert.Equal(t, "my-k8s-cluster", config.Workflow.Context)
	assert.Equal(t, "ralph-workflows", config.Workflow.Namespace)
}

func TestLoadConfig_WorkflowEnvSecretKeyRef(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `workflow:
  env:
    LOG_LEVEL: debug
    API_KEY:
      secretKeyRef:
        name: my-secret
        key: api-key
    PORT: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, config.Workflow.Env, 3)

	literal := config.Workflow.Env["LOG_LEVEL"]
	assert.Equal(t, "debug", literal.Value)
	assert.Nil(t, literal.SecretKeyRef, "literal env entry must have nil SecretKeyRef")

	port := config.Workflow.Env["PORT"]
	assert.Equal(t, "8080", port.Value, "non-string scalar must coerce to string")
	assert.Nil(t, port.SecretKeyRef, "literal env entry must have nil SecretKeyRef")

	secret := config.Workflow.Env["API_KEY"]
	assert.Empty(t, secret.Value, "secret env entry must have empty Value")
	require.NotNil(t, secret.SecretKeyRef, "secret env entry must have SecretKeyRef set")
	assert.Equal(t, "my-secret", secret.SecretKeyRef.Name)
	assert.Equal(t, "api-key", secret.SecretKeyRef.Key)
}

func TestLoadConfig_WorkflowEnvSecretKeyRefValidation(t *testing.T) {
	tests := []struct {
		name    string
		envYAML string
		errMsg  string
	}{
		{
			name:    "missing name",
			envYAML: "    API_KEY:\n      secretKeyRef:\n        key: api-key\n",
			errMsg:  `env "API_KEY": secretKeyRef requires a name`,
		},
		{
			name:    "missing key",
			envYAML: "    API_KEY:\n      secretKeyRef:\n        name: my-secret\n",
			errMsg:  `env "API_KEY": secretKeyRef requires a key`,
		},
		{
			name:    "empty secretKeyRef",
			envYAML: "    API_KEY:\n      secretKeyRef: {}\n",
			errMsg:  `env "API_KEY": secretKeyRef requires a name`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			ralphDir := filepath.Join(tmpDir, ".ralph")
			require.NoError(t, os.Mkdir(ralphDir, 0755))

			configContent := "workflow:\n  env:\n" + tt.envYAML
			configPath := filepath.Join(ralphDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

			t.Chdir(tmpDir)

			_, err := LoadConfig()
			require.Error(t, err, "LoadConfig() expected error")
			assert.Contains(t, err.Error(), "invalid workflow env")
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestLoadConfig_WorkflowEnvInvalidStructure(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		errMsg        string
		wantLine      string
	}{
		{
			name: "mapping without secretKeyRef",
			configContent: `workflow:
  env:
    API_KEY:
      something: else
`,
			errMsg: "must be a string or a mapping with a secretKeyRef",
		},
		{
			name: "sequence value",
			configContent: `workflow:
  env:
    API_KEY:
      - one
      - two
`,
			errMsg: "must be a string or a mapping with a secretKeyRef",
		},
		{
			name: "null value",
			configContent: `workflow:
  env:
    API_KEY:
`,
			errMsg:   "must be a string or a mapping with a secretKeyRef",
			wantLine: "line 3",
		},
		{
			name: "anchored null",
			configContent: `nothing: &n
workflow:
  env:
    API_KEY: *n
`,
			errMsg: "must be a string or a mapping with a secretKeyRef",
		},
		{
			name: "secretKeyRef not a mapping",
			configContent: `workflow:
  env:
    API_KEY:
      secretKeyRef:
        - one
`,
			errMsg: "secretKeyRef must be a mapping with a name and key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			ralphDir := filepath.Join(tmpDir, ".ralph")
			require.NoError(t, os.Mkdir(ralphDir, 0755))

			configPath := filepath.Join(ralphDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.configContent), 0644))

			t.Chdir(tmpDir)

			_, err := LoadConfig()
			require.Error(t, err, "LoadConfig() expected error")
			assert.Contains(t, err.Error(), "failed to parse config YAML")
			assert.Contains(t, err.Error(), tt.errMsg)
			if tt.wantLine != "" {
				assert.Contains(t, err.Error(), tt.wantLine)
			}
		})
	}
}

func TestEnvVarYAMLRoundTrip(t *testing.T) {
	input := map[string]EnvVar{
		"LOG_LEVEL": {Value: "debug"},
		"API_KEY":   {SecretKeyRef: &SecretKeyRef{Name: "my-secret", Key: "api-key"}},
	}

	out, err := yaml.Marshal(input)
	require.NoError(t, err, "yaml.Marshal unexpected error")

	var got map[string]EnvVar
	require.NoError(t, yaml.Unmarshal(out, &got), "yaml.Unmarshal unexpected error")

	require.Len(t, got, 2)

	literal, ok := got["LOG_LEVEL"]
	require.True(t, ok, "LOG_LEVEL missing after round trip")
	assert.Equal(t, "debug", literal.Value)
	assert.Nil(t, literal.SecretKeyRef, "literal entry must round-trip with nil SecretKeyRef")

	secret, ok := got["API_KEY"]
	require.True(t, ok, "API_KEY missing after round trip")
	assert.Empty(t, secret.Value, "secret entry must round-trip with empty Value")
	require.NotNil(t, secret.SecretKeyRef, "secret entry must round-trip with SecretKeyRef")
	assert.Equal(t, "my-secret", secret.SecretKeyRef.Name)
	assert.Equal(t, "api-key", secret.SecretKeyRef.Key)

	data := string(out)
	nameIdx := strings.Index(data, "name: my-secret")
	keyIdx := strings.Index(data, "key: api-key")
	require.True(t, nameIdx >= 0, "marshaled YAML missing name: my-secret")
	require.True(t, keyIdx >= 0, "marshaled YAML missing key: api-key")
	assert.Less(t, nameIdx, keyIdx, "secretKeyRef must marshal name before key")
}

func TestLoadConfig_WithPartialWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `workflow:
  image:
    repository: my-registry/ralph
  context: dev-cluster
  `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	// Verify partial config loads correctly
	assert.Equal(t, "my-registry/ralph", config.Workflow.Image.Repository)
	assert.Equal(t, "", config.Workflow.Image.Tag)
	assert.Equal(t, "dev-cluster", config.Workflow.Context)

	// Verify optional fields are empty/nil
	assert.Len(t, config.Workflow.ConfigMaps, 0)
	assert.Len(t, config.Workflow.Secrets, 0)
	assert.Len(t, config.Workflow.Env, 0)
}

func TestLoadConfig_WithWorkflowLabels(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `workflow:
  labels:
    environment: production
    team: platform
    app.kubernetes.io/name: ralph
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, config.Workflow.Labels, 3)
	assert.Equal(t, "production", config.Workflow.Labels["environment"])
	assert.Equal(t, "platform", config.Workflow.Labels["team"])
	assert.Equal(t, "ralph", config.Workflow.Labels["app.kubernetes.io/name"])
}

func TestLoadConfig_WithoutWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
defaultBranch: main
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	// Verify workflow config exists but is empty
	assert.Equal(t, "", config.Workflow.Image.Repository)
	assert.Equal(t, "", config.Workflow.Image.Tag)
	assert.Equal(t, "", config.Workflow.Context)
	assert.Equal(t, "", config.Workflow.Namespace)
}

func TestApplyDefaults_Model(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `model: ""
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "deepseek/deepseek-chat", config.Model)
}

func TestApplyDefaults_AppFields(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `app:
  name: ""
  id: ""
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "ralph-zon", config.App.Name)
	assert.Equal(t, "2966665", config.App.ID)
}

func TestApplyDefaults_ServiceTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `services:
  - name: svc1
    command: echo
    timeout: 0
  - name: svc2
    command: echo
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, config.Services, 2)
	assert.Equal(t, 30, config.Services[0].Timeout)
	assert.Equal(t, 30, config.Services[1].Timeout)
}

func TestApplyDefaults_DoesNotOverwriteNonZeroValues(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 5
defaultBranch: develop
model: anthropic/claude-3-sonnet
agent: code-reviewer
app:
  name: my-app
  id: 1234567
services:
  - name: svc1
    command: echo
    timeout: 60
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "develop", config.DefaultBranch)
	assert.Equal(t, "anthropic/claude-3-sonnet", config.Model)
	assert.Equal(t, "code-reviewer", config.Agent)
	assert.Equal(t, "my-app", config.App.Name)
	assert.Equal(t, "1234567", config.App.ID)
	assert.Equal(t, 60, config.Services[0].Timeout)
}

func TestLoadConfig_CommentInstructionsFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	customCommentInstructions := "Custom comment instructions for PR comments"
	commentInstructionsPath := filepath.Join(ralphDir, "comment-instructions.md")
	require.NoError(t, os.WriteFile(commentInstructionsPath, []byte(customCommentInstructions), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, customCommentInstructions, config.CommentInstructions)
}

func TestLoadConfig_DefaultInstructionsWhenFilesNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.NotEmpty(t, config.CommentInstructions, "CommentInstructions is empty, expected default instructions")
	assert.Empty(t, config.Instructions, "Instructions must stay unset so the prompt supplies its default steps")
}

// Item test: the ralph config no longer reads `.ralph/merge-instructions.md` —
// the file is ignored, so the config keeps the default comment instructions and
// no development instructions.
func TestLoadConfig_MergeInstructionsFileIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("maxIterations: 3\n"), 0644))

	mergeInstructionsPath := filepath.Join(ralphDir, "merge-instructions.md")
	require.NoError(t, os.WriteFile(mergeInstructionsPath, []byte("Custom merge instructions for merging PRs"), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Empty(t, config.Instructions)
	assert.NotContains(t, config.CommentInstructions, "Custom merge instructions", "the merge instructions file must not be read")
	assert.Contains(t, config.CommentInstructions, "# Comment Instructions", "comment instructions must still fall back to the embedded default")
}

// Item test: the embedded default merge instructions file is gone and no longer
// embedded by the config loader — loadInstructions never reads
// `.ralph/merge-instructions.md`, so the file has no effect on what is loaded.
func TestLoadInstructions_MergeInstructionsFileIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(configDir, 0755))

	instructions, commentInstructions := loadInstructions(configDir)

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "merge-instructions.md"), []byte("Custom merge instructions"), 0644))

	withMerge, withMergeComment := loadInstructions(configDir)
	assert.Equal(t, instructions, withMerge, "merge-instructions.md must not be read into development instructions")
	assert.Equal(t, commentInstructions, withMergeComment, "merge-instructions.md must not be read into comment instructions")
	assert.Contains(t, commentInstructions, "# Comment Instructions")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	invalidYAML := `maxIterations: [this is not valid yaml
  - broken
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(invalidYAML), 0644))

	t.Chdir(tmpDir)

	_, err := LoadConfig()
	require.Error(t, err, "LoadConfig() expected error for invalid YAML")
}

func TestFindConfigDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ralph directory
	ralphDir := filepath.Join(tmpDir, ".ralph")
	err := os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err)

	// Test finding from the same directory
	found, err := FindConfigDir(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, ralphDir, found)

	// Test finding from a subdirectory
	subDir := filepath.Join(tmpDir, "a", "b", "c")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	found, err = FindConfigDir(subDir)
	require.NoError(t, err)
	assert.Equal(t, ralphDir, found)

	// Test not finding it
	otherDir := t.TempDir()
	_, err = FindConfigDir(otherDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestValidateReviewConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ReviewConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with text item",
			config: &ReviewConfig{
				Model: "google/gemini-2.5-pro",
				Items: []ReviewItem{
					{Text: "camel case vars"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with file item",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{File: "docs/standards/deep-modules.md"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with url item",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{URL: "https://raw.githubusercontent.com/zon/code/refs/heads/main/README.md"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple items",
			config: &ReviewConfig{
				Model: "google/gemini-2.5-pro",
				Items: []ReviewItem{
					{Text: "camel case vars"},
					{File: "docs/standards/deep-modules.md"},
					{URL: "https://raw.githubusercontent.com/zon/code/refs/heads/main/README.md"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty items list",
			config: &ReviewConfig{
				Items: []ReviewItem{},
			},
			wantErr: true,
			errMsg:  "review must have at least one item",
		},
		{
			name: "item with no source",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{},
				},
			},
			wantErr: true,
			errMsg:  "review item 0 must have one of text, file, or url set",
		},
		{
			name: "item with multiple sources",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{Text: "some text", File: "somefile.md"},
				},
			},
			wantErr: true,
			errMsg:  "review item 0 must have exactly one of text, file, or url set",
		},
		{
			name: "item with all three sources",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{Text: "some text", File: "somefile.md", URL: "https://example.com"},
				},
			},
			wantErr: true,
			errMsg:  "review item 0 must have exactly one of text, file, or url set",
		},
		{
			name: "valid item with domain-function loop",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{File: "docs/review/domain-function.md", Loop: "domain-function"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid item with text and loop",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{Text: "Review functions", Loop: "domain-function"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid item with url and loop",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{URL: "https://example.com/review.md", Loop: "domain-function"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid loop type",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{Text: "some text", Loop: "unknown-type"},
				},
			},
			wantErr: true,
			errMsg:  `review item 0 has invalid loop type "unknown-type"; valid types are: domain-function`,
		},
		{
			name: "item with empty loop is valid",
			config: &ReviewConfig{
				Items: []ReviewItem{
					{Text: "some text", Loop: ""},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewConfig(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig_WithReviewConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `review:
  model: google/gemini-2.5-pro
  items:
  - text: camel case vars
  - file: docs/standards/deep-modules.md
  - url: https://raw.githubusercontent.com/zon/code/refs/heads/main/README.md
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "google/gemini-2.5-pro", config.Review.Model)
	require.Len(t, config.Review.Items, 3)
	assert.Equal(t, "camel case vars", config.Review.Items[0].Text)
	assert.Equal(t, "docs/standards/deep-modules.md", config.Review.Items[1].File)
	assert.Equal(t, "https://raw.githubusercontent.com/zon/code/refs/heads/main/README.md", config.Review.Items[2].URL)
}

func TestLoadConfig_InvalidReviewConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `review:
  items:
  - text: some text
    file: somefile.md
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	_, err := LoadConfig()
	require.Error(t, err, "LoadConfig() expected error for invalid review config")
	assert.Contains(t, err.Error(), "invalid review config")
}

func TestLoadConfig_ReviewConfigEmptyItems(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `review:
  items: []
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	_, err := LoadConfig()
	require.Error(t, err, "LoadConfig() expected error for empty review items")
	assert.Contains(t, err.Error(), "review must have at least one item")
}

func TestLoadConfigFromPath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing file returns empty config", func(t *testing.T) {
		config, err := loadConfigFromPath(filepath.Join(tmpDir, "nonexistent.yaml"))
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "", config.ConfigPath)
	})

	t.Run("valid YAML returns parsed config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.yaml")
		content := `defaultBranch: develop
model: anthropic/claude-3-sonnet`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := loadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, configPath, config.ConfigPath)
		assert.Equal(t, "develop", config.DefaultBranch)
		assert.Equal(t, "anthropic/claude-3-sonnet", config.Model)
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		content := `defaultBranch: [invalid`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := loadConfigFromPath(configPath)
		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to parse config YAML")
	})

	t.Run("file exists but unreadable returns error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test when running as root")
		}
		configPath := filepath.Join(tmpDir, "unreadable.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("maxIterations: 1"), 0000))
		defer os.Chmod(configPath, 0644)

		config, err := loadConfigFromPath(configPath)
		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestConfigExtraIterationsOmittedWhenNil(t *testing.T) {
	cfg := &RalphConfig{}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "extraIterations")
}

func TestConfigExtraIterationsSerializedWhenSet(t *testing.T) {
	v := 5
	cfg := &RalphConfig{ExtraIterations: &v}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(out), "extraIterations: 5")
}

func TestConfigExtraIterations_UnsetReturnsNil(t *testing.T) {
	cfg := &RalphConfig{}
	resolved := cfg.ExtraIterations
	assert.Nil(t, resolved, "expected nil when ExtraIterations is unset")
}

func TestConfigExtraIterations_ConfigValueReturnedWhenSet(t *testing.T) {
	v := 3
	cfg := &RalphConfig{ExtraIterations: &v}
	resolved := cfg.ExtraIterations
	require.NotNil(t, resolved)
	assert.Equal(t, 3, *resolved)
}

func TestLoadInstructions(t *testing.T) {
	t.Run("all files missing leaves development instructions to the prompt", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".ralph")
		require.NoError(t, os.Mkdir(configDir, 0755))

		instructions, commentInstructions := loadInstructions(configDir)
		assert.Empty(t, instructions)
		assert.Contains(t, commentInstructions, "# Comment Instructions")
	})

	t.Run("custom instructions loaded from files", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".ralph")
		require.NoError(t, os.Mkdir(configDir, 0755))

		customInstructions := "Custom instructions"
		customComment := "Custom comment instructions"

		require.NoError(t, os.WriteFile(filepath.Join(configDir, "instructions.md"), []byte(customInstructions), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "comment-instructions.md"), []byte(customComment), 0644))

		instructions, commentInstructions := loadInstructions(configDir)
		assert.Equal(t, customInstructions, instructions)
		assert.Equal(t, customComment, commentInstructions)
	})

	t.Run("mixed presence uses defaults for missing files", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".ralph")
		require.NoError(t, os.Mkdir(configDir, 0755))

		customInstructions := "Custom instructions"
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "instructions.md"), []byte(customInstructions), 0644))
		// comment-instructions.md missing

		instructions, commentInstructions := loadInstructions(configDir)
		assert.Equal(t, customInstructions, instructions)
		assert.Contains(t, commentInstructions, "# Comment Instructions")
	})
}
