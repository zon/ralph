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
