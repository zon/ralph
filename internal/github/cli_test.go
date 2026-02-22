package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubRemoteURL(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		wantName    string
		wantOwner   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "SSH format with .git suffix",
			remoteURL: "git@github.com:acme/my-repo.git",
			wantName:  "my-repo",
			wantOwner: "acme",
		},
		{
			name:      "SSH format without .git suffix",
			remoteURL: "git@github.com:acme/my-repo",
			wantName:  "my-repo",
			wantOwner: "acme",
		},
		{
			name:      "HTTPS format with .git suffix",
			remoteURL: "https://github.com/acme/my-repo.git",
			wantName:  "my-repo",
			wantOwner: "acme",
		},
		{
			name:      "HTTPS format without .git suffix",
			remoteURL: "https://github.com/acme/my-repo",
			wantName:  "my-repo",
			wantOwner: "acme",
		},
		{
			name:        "non-GitHub URL",
			remoteURL:   "https://gitlab.com/acme/my-repo.git",
			wantErr:     true,
			errContains: "not a GitHub repository URL",
		},
		{
			name:        "empty URL",
			remoteURL:   "",
			wantErr:     true,
			errContains: "remote.origin.url is empty",
		},
		{
			name:        "invalid path",
			remoteURL:   "git@github.com:nodash",
			wantErr:     true,
			errContains: "invalid repository path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, owner, err := parseGitHubRemoteURL(tc.remoteURL)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, name)
			assert.Equal(t, tc.wantOwner, owner)
		})
	}
}
