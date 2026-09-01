package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/setremote"
	"github.com/zon/ralph/internal/output"
)

func TestSetRemoteCmd_FlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantKey     string
		wantToken   string
		wantContext string
		wantNs      string
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantKey: "",
		},
		{
			name:      "github-key flag",
			args:      []string{"--github-key", "/path/to/key.pem"},
			wantKey:   "/path/to/key.pem",
			wantToken: "",
		},
		{
			name:      "github-token flag",
			args:      []string{"--github-token", "ghp_test_token"},
			wantKey:   "",
			wantToken: "ghp_test_token",
		},
		{
			name:        "all flags",
			args:        []string{"--github-token", "ghp_test_token", "--context", "staging", "--namespace", "argo"},
			wantToken:   "ghp_test_token",
			wantContext: "staging",
			wantNs:      "argo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &SetRemoteCmd{}

			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.wantKey, cmd.GithubKey)
			assert.Equal(t, tt.wantToken, cmd.GithubToken)
			assert.Equal(t, tt.wantContext, cmd.Context)
			assert.Equal(t, tt.wantNs, cmd.Namespace)
		})
	}
}

func TestSetRemoteGitHubClientConfigureTokenWritesSecret(t *testing.T) {
	var capturedSecretName, capturedNamespace, capturedContext string
	var capturedData map[string]string

	k8sClient := &k8s.MockClient{
		CreateOrUpdateSecretFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			capturedSecretName = name
			capturedNamespace = namespace
			capturedContext = kubeContext
			capturedData = data
			return nil
		},
	}

	out := output.NewClient(io.Discard, io.Discard, false)
	client := &setremoteGitHubClient{ctx: context.Background(), k8sClient: k8sClient, out: out}

	err := client.ConfigureToken(setremote.K8sContext{Name: "staging", Namespace: "argo"}, "ghp_test_token")
	require.NoError(t, err)
	assert.Equal(t, k8s.GitHubSecretName, capturedSecretName)
	assert.Equal(t, "argo", capturedNamespace)
	assert.Equal(t, "staging", capturedContext)
	assert.Equal(t, map[string]string{"token": "ghp_test_token"}, capturedData)
}

func TestSetRemoteGitHubClientTokenFromGHCli(t *testing.T) {
	t.Run("returns gh auth token", func(t *testing.T) {
		writeFakeGHCLIScript(t, `printf 'ghp_cli_token\n'`)
		client := &setremoteGitHubClient{}
		assert.Equal(t, "ghp_cli_token", client.TokenFromGHCli())
	})

	t.Run("returns empty when gh not authenticated", func(t *testing.T) {
		writeFakeGHCLIScript(t, `exit 1`)
		client := &setremoteGitHubClient{}
		assert.Empty(t, client.TokenFromGHCli())
	})
}

func TestSetRemoteGitHubClientTokenFromEnv(t *testing.T) {
	t.Run("returns environment token", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "ghp_env_token")
		client := &setremoteGitHubClient{}
		assert.Equal(t, "ghp_env_token", client.TokenFromEnv())
	})

	t.Run("returns empty when GITHUB_TOKEN is unset", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		client := &setremoteGitHubClient{}
		assert.Empty(t, client.TokenFromEnv())
	})
}

func writeFakeGHCLIScript(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755))
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)
}

func TestSetRemoteGitHubClientConfigureTokenPropagatesError(t *testing.T) {
	k8sClient := &k8s.MockClient{
		CreateOrUpdateSecretFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			return assert.AnError
		},
	}

	out := output.NewClient(io.Discard, io.Discard, false)
	client := &setremoteGitHubClient{ctx: context.Background(), k8sClient: k8sClient, out: out}

	err := client.ConfigureToken(setremote.K8sContext{Name: "staging", Namespace: "argo"}, "ghp_test_token")
	require.Error(t, err)
}
