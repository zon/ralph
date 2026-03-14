package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/config"
)

func TestBuildDebugScript(t *testing.T) {
	tests := []struct {
		name            string
		verbose         bool
		noServices      bool
		expectedCommand string
	}{
		{
			name:            "no flags",
			verbose:         false,
			noServices:      false,
			expectedCommand: `ralph workflow`,
		},
		{
			name:            "no services",
			verbose:         false,
			noServices:      true,
			expectedCommand: `ralph workflow --no-services`,
		},
		{
			name:            "verbose",
			verbose:         true,
			noServices:      false,
			expectedCommand: `ralph workflow --verbose`,
		},
		{
			name:            "verbose and no services",
			verbose:         true,
			noServices:      true,
			expectedCommand: `ralph workflow --verbose --no-services`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.RalphConfig{
				Workflow: config.WorkflowConfig{},
			}
			script := buildDebugScript(tt.verbose, tt.noServices, "main", cfg)

			expectedElements := []string{
				"#!/bin/sh",
				"set -e",
				"ralph workflow",
				"Execution complete",
				tt.expectedCommand,
			}

			for _, element := range expectedElements {
				assert.Contains(t, script, element, "debug script should contain expected element")
			}
		})
	}
}

func TestBuildDebugScript_DebugBranch(t *testing.T) {
	cfg := &config.RalphConfig{
		Workflow: config.WorkflowConfig{},
	}
	script := buildDebugScript(false, false, "my-debug-branch", cfg)

	expectedElements := []string{
		"git clone -b \"my-debug-branch\" https://github.com/zon/ralph.git /workspace/ralph",
		"go run ./cmd/ralph/main.go",
		"ralph workflow",
	}
	for _, element := range expectedElements {
		assert.Contains(t, script, element, "debug script (debug branch) should contain expected element")
	}

	assert.NotContains(t, script, "command ralph", "debug script (debug branch) should not use 'command ralph' fallback")
}

func TestBuildCommentScript(t *testing.T) {
	tests := []struct {
		name            string
		verbose         bool
		expectedCommand string
	}{
		{
			name:            "no flags",
			verbose:         false,
			expectedCommand: `ralph comment "$COMMENT_BODY" --repo "$GITHUB_REPO_OWNER/$GITHUB_REPO_NAME" --branch "$PROJECT_BRANCH" --pr "$PR_NUMBER"`,
		},
		{
			name:            "verbose",
			verbose:         true,
			expectedCommand: `ralph comment "$COMMENT_BODY" --repo "$GITHUB_REPO_OWNER/$GITHUB_REPO_NAME" --branch "$PROJECT_BRANCH" --pr "$PR_NUMBER" --verbose`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := buildCommentScript(tt.verbose, false)

			expectedElements := []string{
				"#!/bin/sh",
				"set -e",
				"git clone",
				"GIT_REPO_URL",
				"GIT_BRANCH",
				"ralph set-github-token",
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"auth.json",
				"ralph comment",
				"COMMENT_BODY",
				"PR_NUMBER",
				"opencode stats",
				tt.expectedCommand,
			}

			for _, element := range expectedElements {
				assert.Contains(t, script, element, "comment script should contain expected element")
			}
		})
	}
}

func TestBuildMergeScript(t *testing.T) {
	tests := []struct {
		name             string
		expectStrings    []string
		notExpectStrings []string
	}{
		{
			name: "merge script",
			expectStrings: []string{
				"#!/bin/sh",
				"set -e",
				"git clone",
				"GIT_REPO_URL",
				"GIT_BRANCH",
				"PR_BRANCH",
				"PR_NUMBER",
				"ralph set-github-token",
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"ralph merge",
				"--local",
				"--pr",
			},
			notExpectStrings: []string{
				"mkdir -p ~/.ssh",
				"ssh-privatekey",
				"ssh-keyscan",
				"passing: false",
				"gh pr merge",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := buildMergeScript()
			for _, s := range tt.expectStrings {
				assert.Contains(t, script, s, "merge script should contain expected element")
			}
			for _, s := range tt.notExpectStrings {
				assert.NotContains(t, script, s, "merge script should not contain unexpected element")
			}
		})
	}
}

func TestBuildVolumeMounts_WorkspacePrefix(t *testing.T) {
	cfg := &config.RalphConfig{
		Workflow: config.WorkflowConfig{
			ConfigMaps: []config.ConfigMapMount{
				{Name: "my-config", DestFile: "config/main.yaml"},
				{Name: "my-config-dir", DestDir: "config/extra"},
			},
			Secrets: []config.SecretMount{
				{Name: "my-secret", DestFile: "config/secrets.yaml"},
				{Name: "my-secret-dir", DestDir: "config/auth"},
			},
		},
	}

	mounts := buildVolumeMounts(cfg)

	expected := map[string]string{
		"my-config-0":   "/workspace/config/main.yaml",
		"my-config-dir": "/workspace/config/extra",
		"my-secret-1":   "/workspace/config/secrets.yaml",
		"my-secret-dir": "/workspace/config/auth",
	}

	for _, mount := range mounts {
		name, _ := mount["name"].(string)
		mountPath, _ := mount["mountPath"].(string)
		if want, ok := expected[name]; ok {
			assert.Equal(t, want, mountPath, "mount %q should have correct mountPath", name)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-config", "my-config"},
		{"my_config", "my-config"},
		{"my.config", "my-config"},
		{"MyConfig", "myconfig"},
		{"my_config.map", "my-config-map"},
		{"my/config", "my-config"},
		{"my/branch.name", "my-branch-name"},
		{"MY_BRANCH", "my-branch"},
		{"feature/123-add-thing", "feature-123-add-thing"},
		{"feature@special!", "feature-special"},
		{"123-branch", "branch-123-branch"},
		{"---test---", "test"},
		{"", "default"},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		assert.Equal(t, tt.expected, result, "sanitizeName should return expected value")
	}
}

func TestBuildConfigMapVolumeMount(t *testing.T) {
	tests := []struct {
		name     string
		destFile string
		destDir  string
		index    int
		check    func(t *testing.T, mount map[string]interface{})
	}{
		{
			name:     "my-config",
			destFile: "config/main.yaml",
			destDir:  "",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "my-config-0", mount["name"], "name should match")
				assert.Equal(t, "/workspace/config/main.yaml", mount["mountPath"], "mountPath should match")
				assert.Equal(t, "main.yaml", mount["subPath"], "subPath should match")
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "config/extra",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "my-config", mount["name"], "name should match")
				assert.Equal(t, "/workspace/config/extra", mount["mountPath"], "mountPath should match")
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "/configmaps/my-config", mount["mountPath"], "mountPath should match")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := buildConfigMapVolumeMount(tt.name, tt.destFile, tt.destDir, tt.index)
			tt.check(t, mount)
		})
	}
}

func TestBuildSecretVolumeMount(t *testing.T) {
	mount := buildSecretVolumeMount("my-secret", "secrets.yaml", "", 0)
	assert.Equal(t, "my-secret-0", mount["name"], "name should match")
	assert.Equal(t, "/workspace/secrets.yaml", mount["mountPath"], "mountPath should match")
}

func TestBuildCredentialMounts(t *testing.T) {
	mounts := buildCredentialMounts()
	assert.Len(t, mounts, 3, "should have 3 credential mounts")
	assert.Equal(t, "github-credentials", mounts[0]["name"], "first mount name should match")
	assert.Equal(t, "opencode-credentials", mounts[1]["name"], "second mount name should match")
	assert.Equal(t, "pulumi-credentials", mounts[2]["name"], "third mount name should match")
}

func TestBuildCredentialVolumes(t *testing.T) {
	volumes := buildCredentialVolumes()
	assert.Len(t, volumes, 3, "should have 3 credential volumes")
	assert.Equal(t, "github-credentials", volumes[0]["name"], "first volume name should match")
	assert.Equal(t, "opencode-credentials", volumes[1]["name"], "second volume name should match")
	assert.Equal(t, "pulumi-credentials", volumes[2]["name"], "third volume name should match")

	// Verify Pulumi secret is optional
	pulumiVol, ok := volumes[2]["secret"].(map[string]interface{})
	assert.True(t, ok, "pulumi volume should have secret map")
	assert.Equal(t, true, pulumiVol["optional"], "pulumi secret should be optional")
}

func TestBuildConfigMapVolume(t *testing.T) {
	vol := buildConfigMapVolume("my-config", "config/main.yaml", 0)
	assert.Equal(t, "my-config-0", vol["name"], "name should match")

	vol = buildConfigMapVolume("my-config", "", 0)
	assert.Equal(t, "my-config", vol["name"], "name should match")
}

func TestBuildSecretVolume(t *testing.T) {
	vol := buildSecretVolume("my-secret", "secrets.yaml", 0)
	assert.Equal(t, "my-secret-0", vol["name"], "name should match")

	vol = buildSecretVolume("my-secret", "", 0)
	assert.Equal(t, "my-secret", vol["name"], "name should match")
}
