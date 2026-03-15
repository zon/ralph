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
		wantOwner   string
		wantName    string
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
		{
			name:      "HTTPS with x-access-token credentials",
			remoteURL: "https://x-access-token:ghp_abc123@github.com/owner/repo.git",
			wantName:  "repo",
			wantOwner: "owner",
		},
		{
			name:      "HTTPS with extra path segments",
			remoteURL: "https://github.com/owner/repo/tree/main/subdir",
			wantName:  "repo",
			wantOwner: "owner",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, err := ParseGitHubRemoteURL(tc.remoteURL)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantOwner, repo.Owner)
			assert.Equal(t, tc.wantName, repo.Name)
		})
	}
}

func TestParseSSHKeyListOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		title       string
		wantKeyID   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "returns key ID when title matches",
			output:    `ralph-myrepo ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbCdEfGhIjKlMnOpQrStUvWxYz123456789 2025-02-15T12:00:00Z 1234567890 authentication`,
			title:     "ralph-myrepo",
			wantKeyID: "1234567890",
		},
		{
			name:      "returns empty string when no title matches",
			output:    `other-key ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbCdEfGhIjKlMnOpQrStUvWxYz123456789 2025-02-15T12:00:00Z 1234567890 authentication`,
			title:     "ralph-myrepo",
			wantKeyID: "",
		},
		{
			name: "skips warning lines",
			output: `warning: could not read SSH keys
ralph-myrepo ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbCdEfGhIjKlMnOpQrStUvWxYz123456789 2025-02-15T12:00:00Z 1234567890 authentication`,
			title:     "ralph-myrepo",
			wantKeyID: "1234567890",
		},
		{
			name: "skips empty lines",
			output: `
ralph-myrepo ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbCdEfGhIjKlMnOpQrStUvWxYz123456789 2025-02-15T12:00:00Z 1234567890 authentication`,
			title:     "ralph-myrepo",
			wantKeyID: "1234567890",
		},
		{
			name: "skips whitespace-only lines",
			output: `   
ralph-myrepo ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbCdEfGhIjKlMnOpQrStUvWxYz123456789 2025-02-15T12:00:00Z 1234567890 authentication`,
			title:     "ralph-myrepo",
			wantKeyID: "1234567890",
		},
		{
			name: "skips lines with fewer than 5 fields",
			output: `not-enough-fields
ralph-myrepo ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbCdEfGhIjKlMnOpQrStUvWxYz123456789 2025-02-15T12:00:00Z 1234567890 authentication`,
			title:     "ralph-myrepo",
			wantKeyID: "1234567890",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keyID, err := parseSSHKeyListOutput(tc.output, tc.title)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantKeyID, keyID)
		})
	}
}
