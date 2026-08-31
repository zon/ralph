package cmd

import (
	"context"
	"io"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/setconfig"
	"github.com/zon/ralph/internal/output"
)

func TestSetConfigCmd_FlagParsing(t *testing.T) {
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
			cmd := &SetConfigCmd{}

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

func TestSetConfigGitHubClientConfigureTokenWritesSecret(t *testing.T) {
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
	client := &setconfigGitHubClient{ctx: context.Background(), k8sClient: k8sClient, out: out}

	err := client.ConfigureToken(setconfig.K8sContext{Name: "staging", Namespace: "argo"}, "ghp_test_token")
	require.NoError(t, err)
	assert.Equal(t, k8s.GitHubSecretName, capturedSecretName)
	assert.Equal(t, "argo", capturedNamespace)
	assert.Equal(t, "staging", capturedContext)
	assert.Equal(t, map[string]string{"token": "ghp_test_token"}, capturedData)
}

func TestSetConfigGitHubClientConfigureTokenPropagatesError(t *testing.T) {
	k8sClient := &k8s.MockClient{
		CreateOrUpdateSecretFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			return assert.AnError
		},
	}

	out := output.NewClient(io.Discard, io.Discard, false)
	client := &setconfigGitHubClient{ctx: context.Background(), k8sClient: k8sClient, out: out}

	err := client.ConfigureToken(setconfig.K8sContext{Name: "staging", Namespace: "argo"}, "ghp_test_token")
	require.Error(t, err)
}
