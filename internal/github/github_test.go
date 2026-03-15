package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePR(t *testing.T) {
	_, _ = CreatePR("Test PR", "Test body", "main", "feature-branch")
}

func TestMakeRepo(t *testing.T) {
	repo := MakeRepo("zon", "ralph")
	assert.Equal(t, "zon", repo.Owner)
	assert.Equal(t, "ralph", repo.Name)
}

func TestCloneURL(t *testing.T) {
	assert.Equal(t, "https://github.com/zon/ralph.git", CloneURL("zon", "ralph"))
}

func TestExtractExistingPRURL(t *testing.T) {
	tests := []struct {
		name     string
		errStr   string
		expected string
	}{
		{
			name:     "extract URL from error message",
			errStr:   "A pull request for branch 'feature-branch' already exists.\nhttps://github.com/zon/ralph/pull/123",
			expected: "https://github.com/zon/ralph/pull/123",
		},
		{
			name:     "no URL in message",
			errStr:   "Some other error occurred",
			expected: "",
		},
		{
			name:     "URL at start of line",
			errStr:   "https://github.com/zon/ralph/pull/456\nSome other text",
			expected: "https://github.com/zon/ralph/pull/456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExistingPRURL(tt.errStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
		err      bool
	}{
		{
			name:     "parse URL from output",
			output:   "https://github.com/zon/ralph/pull/789",
			expected: "https://github.com/zon/ralph/pull/789",
			err:      false,
		},
		{
			name:     "parse URL from multiline output",
			output:   "Pull request created.\nhttps://github.com/zon/ralph/pull/101",
			expected: "https://github.com/zon/ralph/pull/101",
			err:      false,
		},
		{
			name:     "no URL in output",
			output:   "Something else",
			expected: "",
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePRURL(tt.output)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
