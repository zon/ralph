package workflow

import (
	"strings"
	"testing"

	"github.com/zon/ralph/internal/config"
)

func TestBuildRunScript(t *testing.T) {
	tests := []struct {
		name            string
		dryRun          bool
		verbose         bool
		expectedCommand string
	}{
		{
			name:            "no flags",
			dryRun:          false,
			verbose:         false,
			expectedCommand: `ralph_run "$PROJECT_PATH" --local --no-notify`,
		},
		{
			name:            "dry-run only",
			dryRun:          true,
			verbose:         false,
			expectedCommand: `ralph_run "$PROJECT_PATH" --local --dry-run --no-notify`,
		},
		{
			name:            "verbose only",
			dryRun:          false,
			verbose:         true,
			expectedCommand: `ralph_run "$PROJECT_PATH" --local --verbose --no-notify`,
		},
		{
			name:            "both flags",
			dryRun:          true,
			verbose:         true,
			expectedCommand: `ralph_run "$PROJECT_PATH" --local --dry-run --verbose --no-notify`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.RalphConfig{
				Workflow: config.WorkflowConfig{},
			}
			script := buildRunScript(tt.dryRun, tt.verbose, "", cfg)

			expectedElements := []string{
				"#!/bin/sh",
				"set -e",
				"git clone",
				"GIT_REPO_URL",
				"GIT_BRANCH",
				"PROJECT_BRANCH",
				"BASE_BRANCH",
				`ralph set-github-token`,
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"auth.json",
				tt.expectedCommand,
			}

			for _, element := range expectedElements {
				if !strings.Contains(script, element) {
					t.Errorf("run script does not contain expected element: %s", element)
				}
			}
		})
	}
}

func TestBuildRunScript_DebugBranch(t *testing.T) {
	cfg := &config.RalphConfig{
		Workflow: config.WorkflowConfig{},
	}
	script := buildRunScript(false, false, "my-debug-branch", cfg)

	expectedElements := []string{
		"git clone -b \"my-debug-branch\" https://github.com/zon/ralph.git /workspace/ralph",
		"go run ./cmd/ralph/main.go",
		`ralph set-github-token`,
		`ralph_run "$PROJECT_PATH" --local --no-notify`,
	}
	for _, element := range expectedElements {
		if !strings.Contains(script, element) {
			t.Errorf("run script (debug branch) does not contain expected element: %s", element)
		}
	}

	// non-debug path must NOT appear when debug branch is set
	if strings.Contains(script, "command ralph") {
		t.Errorf("run script (debug branch) should not use 'command ralph' fallback")
	}
}

func TestBuildCommentScript(t *testing.T) {
	tests := []struct {
		name            string
		dryRun          bool
		verbose         bool
		expectedCommand string
	}{
		{
			name:            "no flags",
			dryRun:          false,
			verbose:         false,
			expectedCommand: `ralph comment "$COMMENT_BODY" --repo "$GITHUB_REPO_OWNER/$GITHUB_REPO_NAME" --branch "$PROJECT_BRANCH" --pr "$PR_NUMBER" --no-notify`,
		},
		{
			name:            "dry-run",
			dryRun:          true,
			verbose:         false,
			expectedCommand: `ralph comment "$COMMENT_BODY" --repo "$GITHUB_REPO_OWNER/$GITHUB_REPO_NAME" --branch "$PROJECT_BRANCH" --pr "$PR_NUMBER" --dry-run --no-notify`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := buildCommentScript(tt.dryRun, tt.verbose)

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
				tt.expectedCommand,
			}

			for _, element := range expectedElements {
				if !strings.Contains(script, element) {
					t.Errorf("comment script does not contain expected element: %s", element)
				}
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
				if !strings.Contains(script, s) {
					t.Errorf("merge script does not contain expected element: %q", s)
				}
			}
			for _, s := range tt.notExpectStrings {
				if strings.Contains(script, s) {
					t.Errorf("merge script unexpectedly contains: %q", s)
				}
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
			if mountPath != want {
				t.Errorf("mount %q: mountPath = %q, want %q", name, mountPath, want)
			}
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
		if result != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
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
				if mount["name"] != "my-config-0" {
					t.Errorf("name = %v, want my-config-0", mount["name"])
				}
				if mount["mountPath"] != "/workspace/config/main.yaml" {
					t.Errorf("mountPath = %v, want /workspace/config/main.yaml", mount["mountPath"])
				}
				if mount["subPath"] != "main.yaml" {
					t.Errorf("subPath = %v, want main.yaml", mount["subPath"])
				}
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "config/extra",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				if mount["name"] != "my-config" {
					t.Errorf("name = %v, want my-config", mount["name"])
				}
				if mount["mountPath"] != "/workspace/config/extra" {
					t.Errorf("mountPath = %v, want /workspace/config/extra", mount["mountPath"])
				}
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				if mount["mountPath"] != "/configmaps/my-config" {
					t.Errorf("mountPath = %v, want /configmaps/my-config", mount["mountPath"])
				}
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
	if mount["name"] != "my-secret-0" {
		t.Errorf("name = %v, want my-secret-0", mount["name"])
	}
	if mount["mountPath"] != "/workspace/secrets.yaml" {
		t.Errorf("mountPath = %v, want /workspace/secrets.yaml", mount["mountPath"])
	}
}

func TestBuildCredentialMounts(t *testing.T) {
	mounts := buildCredentialMounts()
	if len(mounts) != 2 {
		t.Fatalf("expected 2 credential mounts, got %d", len(mounts))
	}
	if mounts[0]["name"] != "github-credentials" {
		t.Errorf("first mount name = %v, want github-credentials", mounts[0]["name"])
	}
	if mounts[1]["name"] != "opencode-credentials" {
		t.Errorf("second mount name = %v, want opencode-credentials", mounts[1]["name"])
	}
}

func TestBuildCredentialVolumes(t *testing.T) {
	volumes := buildCredentialVolumes()
	if len(volumes) != 2 {
		t.Fatalf("expected 2 credential volumes, got %d", len(volumes))
	}
	if volumes[0]["name"] != "github-credentials" {
		t.Errorf("first volume name = %v, want github-credentials", volumes[0]["name"])
	}
}

func TestBuildConfigMapVolume(t *testing.T) {
	vol := buildConfigMapVolume("my-config", "config/main.yaml", 0)
	if vol["name"] != "my-config-0" {
		t.Errorf("name = %v, want my-config-0", vol["name"])
	}

	vol = buildConfigMapVolume("my-config", "", 0)
	if vol["name"] != "my-config" {
		t.Errorf("name = %v, want my-config", vol["name"])
	}
}

func TestBuildSecretVolume(t *testing.T) {
	vol := buildSecretVolume("my-secret", "secrets.yaml", 0)
	if vol["name"] != "my-secret-0" {
		t.Errorf("name = %v, want my-secret-0", vol["name"])
	}

	vol = buildSecretVolume("my-secret", "", 0)
	if vol["name"] != "my-secret" {
		t.Errorf("name = %v, want my-secret", vol["name"])
	}
}
