package workflow

import (
	"strings"
	"testing"

	"github.com/zon/ralph/internal/config"
)

func TestBuildExecutionScript(t *testing.T) {
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
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --no-notify",
		},
		{
			name:            "dry-run only",
			dryRun:          true,
			verbose:         false,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --dry-run --no-notify",
		},
		{
			name:            "verbose only",
			dryRun:          false,
			verbose:         true,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --verbose --no-notify",
		},
		{
			name:            "both flags",
			dryRun:          true,
			verbose:         true,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --dry-run --verbose --no-notify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.RalphConfig{
				Workflow: config.WorkflowConfig{},
			}
			script := buildExecutionScript(tt.dryRun, tt.verbose, cfg)

			expectedElements := []string{
				"#!/bin/sh",
				"set -e",
				"git clone",
				"GIT_REPO_URL",
				"GIT_BRANCH",
				"PROJECT_BRANCH",
				"BASE_BRANCH",
				"ralph github-token",
				"x-access-token:${GITHUB_TOKEN}@github.com",
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"auth.json",
				tt.expectedCommand,
			}

			for _, element := range expectedElements {
				if !strings.Contains(script, element) {
					t.Errorf("Script does not contain expected element: %s", element)
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
				"ralph github-token",
				"x-access-token:${GITHUB_TOKEN}@github.com",
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"passing: false",
				"rm \"$PROJECT_PATH\"",
				"git add -A",
				"git commit",
				"git push",
				"gh pr merge",
				"--merge",
				"--delete-branch",
			},
			notExpectStrings: []string{
				"mkdir -p ~/.ssh",
				"ssh-privatekey",
				"ssh-keyscan",
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
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
