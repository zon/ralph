package cmd

import (
	"testing"
)

func TestRenderInstructions(t *testing.T) {
	tests := []struct {
		name           string
		tmplText       string
		repo           string
		branch         string
		body           string
		pr             string
		expectedOutput string
	}{
		{
			name:           "replaces CommentBody, PRNumber, PRBranch, RepoOwner, RepoName with provided values",
			tmplText:       "Comment: {{.CommentBody}}\nPR: {{.PRNumber}}\nBranch: {{.PRBranch}}\nOwner: {{.RepoOwner}}\nRepo: {{.RepoName}}",
			repo:           "zon/ralph",
			branch:         "feature/test",
			body:           "Please review this",
			pr:             "123",
			expectedOutput: "Comment: Please review this\nPR: 123\nBranch: feature/test\nOwner: zon\nRepo: ralph",
		},
		{
			name:           "splits repo string on / to populate RepoOwner and RepoName correctly",
			tmplText:       "Owner: {{.RepoOwner}}, Repo: {{.RepoName}}",
			repo:           "myorg/my-repo",
			branch:         "main",
			body:           "",
			pr:             "1",
			expectedOutput: "Owner: myorg, Repo: my-repo",
		},
		{
			name:           "returns raw template text when template contains invalid Go template syntax",
			tmplText:       "Invalid: {{.InvalidField",
			repo:           "owner/repo",
			branch:         "main",
			body:           "test",
			pr:             "1",
			expectedOutput: "Invalid: {{.InvalidField",
		},
		{
			name:           "handles repo string without a / by leaving RepoOwner and RepoName as empty strings",
			tmplText:       "Owner: '{{.RepoOwner}}', Repo: '{{.RepoName}}'",
			repo:           "invalid-repo",
			branch:         "main",
			body:           "test",
			pr:             "1",
			expectedOutput: "Owner: '', Repo: ''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderInstructions(tt.tmplText, tt.repo, tt.branch, tt.body, tt.pr)
			if result != tt.expectedOutput {
				t.Errorf("expected %q, got %q", tt.expectedOutput, result)
			}
		})
	}
}

func TestProjectFileFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "converts ralph/my-feature to projects/my-feature.yaml",
			branch:   "ralph/my-feature",
			expected: "projects/my-feature.yaml",
		},
		{
			name:     "converts branch without ralph/ prefix by replacing slashes with dashes",
			branch:   "feature/thing",
			expected: "projects/feature-thing.yaml",
		},
		{
			name:     "handles branch name with no slashes",
			branch:   "my-feature",
			expected: "projects/my-feature.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := projectFileFromBranch(tt.branch)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
