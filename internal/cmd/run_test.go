package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveBaseBranch(t *testing.T) {
	tests := []struct {
		name          string
		baseFlag      string
		currentBranch string
		projectBranch string
		defaultBranch string
		expectedBase  string
	}{
		{
			name:          "base flag provided - use explicit base",
			baseFlag:      "develop",
			currentBranch: "feature-branch",
			projectBranch: "my-feature",
			defaultBranch: "main",
			expectedBase:  "develop",
		},
		{
			name:          "no base flag - current branch different from project - use current branch",
			baseFlag:      "",
			currentBranch: "feature-branch",
			projectBranch: "my-feature",
			defaultBranch: "main",
			expectedBase:  "feature-branch",
		},
		{
			name:          "no base flag - current branch same as project - use default branch",
			baseFlag:      "",
			currentBranch: "my-feature",
			projectBranch: "my-feature",
			defaultBranch: "main",
			expectedBase:  "main",
		},
		{
			name:          "base flag provided even when on project branch",
			baseFlag:      "release-branch",
			currentBranch: "my-feature",
			projectBranch: "my-feature",
			defaultBranch: "main",
			expectedBase:  "release-branch",
		},
		{
			name:          "current branch is main and different from project",
			baseFlag:      "",
			currentBranch: "main",
			projectBranch: "my-feature",
			defaultBranch: "main",
			expectedBase:  "main",
		},
		{
			name:          "custom default branch",
			baseFlag:      "",
			currentBranch: "my-feature",
			projectBranch: "my-feature",
			defaultBranch: "develop",
			expectedBase:  "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveBaseBranch(tt.baseFlag, tt.currentBranch, tt.projectBranch, tt.defaultBranch)
			assert.Equal(t, tt.expectedBase, got, "resolveBaseBranch should return expected base branch")
		})
	}
}

func TestRunFlagsValidate(t *testing.T) {
	tests := []struct {
		name        string
		flags       RunFlags
		wantErr     bool
		errContains string
	}{
		{
			name: "valid flags - local mode",
			flags: RunFlags{
				Local: true,
			},
			wantErr: false,
		},
		{
			name: "valid flags - workflow mode",
			flags: RunFlags{
				Local:  false,
				Follow: true,
			},
			wantErr: false,
		},
		{
			name: "invalid - follow with local",
			flags: RunFlags{
				Follow: true,
				Local:  true,
			},
			wantErr:     true,
			errContains: "--follow flag is not applicable with --local flag",
		},
		{
			name: "invalid - debug with local",
			flags: RunFlags{
				Debug: "some-branch",
				Local: true,
			},
			wantErr:     true,
			errContains: "--debug flag is not applicable with --local flag",
		},
		{
			name: "valid - debug without local",
			flags: RunFlags{
				Debug: "some-branch",
				Local: false,
			},
			wantErr: false,
		},
		{
			name: "valid - all flags set correctly",
			flags: RunFlags{
				Follow: false,
				Local:  false,
				Debug:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
