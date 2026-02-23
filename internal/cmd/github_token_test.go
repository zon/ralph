package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubTokenCmd_FlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantOwner   string
		wantRepo    string
		wantSecrets string
	}{
		{
			name:        "default values",
			args:        []string{},
			wantOwner:   "",
			wantRepo:    "",
			wantSecrets: "/secrets/github",
		},
		{
			name:        "custom owner and repo",
			args:        []string{"--owner", "test-owner", "--repo", "test-repo"},
			wantOwner:   "test-owner",
			wantRepo:    "test-repo",
			wantSecrets: "/secrets/github",
		},
		{
			name:        "custom secrets directory",
			args:        []string{"--secrets-dir", "/custom/secrets"},
			wantOwner:   "",
			wantRepo:    "",
			wantSecrets: "/custom/secrets",
		},
		{
			name:        "all custom flags",
			args:        []string{"--owner", "owner", "--repo", "repo", "--secrets-dir", "/path/to/secrets"},
			wantOwner:   "owner",
			wantRepo:    "repo",
			wantSecrets: "/path/to/secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &GithubTokenCmd{}

			// Use kong to parse the flags
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.wantOwner, cmd.Owner)
			assert.Equal(t, tt.wantRepo, cmd.Repo)
			assert.Equal(t, tt.wantSecrets, cmd.SecretsDir)
		})
	}
}

func TestGithubTokenCmd_Run_MissingCredentials(t *testing.T) {
	// Create a temporary directory for secrets
	tmpDir := t.TempDir()

	cmd := &GithubTokenCmd{
		Owner:      "test-owner",
		Repo:       "test-repo",
		SecretsDir: tmpDir,
	}

	// Test missing app-id
	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read app ID")

	// Create app-id file but not private-key
	appIDPath := filepath.Join(tmpDir, "app-id")
	err = os.WriteFile(appIDPath, []byte("12345"), 0644)
	require.NoError(t, err)

	err = cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read private key")

	// Create empty private-key file
	privateKeyPath := filepath.Join(tmpDir, "private-key")
	err = os.WriteFile(privateKeyPath, []byte(""), 0644)
	require.NoError(t, err)

	err = cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key is empty")
}

func TestGithubTokenCmd_Run_InvalidAppID(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty app-id file
	appIDPath := filepath.Join(tmpDir, "app-id")
	err := os.WriteFile(appIDPath, []byte(""), 0644)
	require.NoError(t, err)

	// Create private-key file with dummy content
	privateKeyPath := filepath.Join(tmpDir, "private-key")
	err = os.WriteFile(privateKeyPath, []byte("-----BEGIN PRIVATE KEY-----\ndummy\n-----END PRIVATE KEY-----\n"), 0644)
	require.NoError(t, err)

	cmd := &GithubTokenCmd{
		Owner:      "test-owner",
		Repo:       "test-repo",
		SecretsDir: tmpDir,
	}

	err = cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app ID is empty")
}

// Note: Testing the actual GitHub API calls would require mocking or integration testing
// which is beyond the scope of unit tests. The actual API integration should be tested
// separately.
