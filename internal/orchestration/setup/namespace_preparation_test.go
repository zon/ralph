package setup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunAppCredentialsPreparedSuccessfully(t *testing.T) {
	var order []string
	var githubCtx, opencodeCtx K8sContext

	cmd := setup.withMocks(
		setup.withContext(&mockContextClient{
			resolveFunc: func(flagContext, flagNamespace string) (K8sContext, error) {
				order = append(order, "resolve")
				return K8sContext{Name: flagContext, Namespace: flagNamespace}, nil
			},
		}),
		setup.withGitHub(&mockGitHubCredentialsClient{
			validateFunc: func(keyPath string) error {
				order = append(order, "validate-github")
				return nil
			},
			configureFunc: func(k8sCtx K8sContext, keyPath string) error {
				order = append(order, "write-github")
				githubCtx = k8sCtx
				return nil
			},
		}),
		setup.withOpenCode(&mockOpenCodeCredentialsClient{
			configureFunc: func(k8sCtx K8sContext) error {
				order = append(order, "write-opencode")
				opencodeCtx = k8sCtx
				return nil
			},
		}),
	)

	err := cmd.Run(Flags{Context: "staging", Namespace: "argo", GithubKey: "/path/to/key.pem"})
	require.NoError(t, err)
	require.Equal(t, []string{"resolve", "validate-github", "write-github", "write-opencode"}, order)
	require.Equal(t, K8sContext{Name: "staging", Namespace: "argo"}, githubCtx)
	require.Equal(t, K8sContext{Name: "staging", Namespace: "argo"}, opencodeCtx)
}

func TestRunTokenCredentialsPreparedSuccessfully(t *testing.T) {
	var order []string
	var githubCtx, opencodeCtx K8sContext

	cmd := setup.withMocks(
		setup.withContext(&mockContextClient{
			resolveFunc: func(flagContext, flagNamespace string) (K8sContext, error) {
				order = append(order, "resolve")
				return K8sContext{Name: flagContext, Namespace: flagNamespace}, nil
			},
		}),
		setup.withGitHub(&mockGitHubCredentialsClient{
			configureTokenFunc: func(k8sCtx K8sContext, token string) error {
				order = append(order, "write-github")
				githubCtx = k8sCtx
				return nil
			},
		}),
		setup.withOpenCode(&mockOpenCodeCredentialsClient{
			configureFunc: func(k8sCtx K8sContext) error {
				order = append(order, "write-opencode")
				opencodeCtx = k8sCtx
				return nil
			},
		}),
	)

	err := cmd.Run(Flags{Context: "staging", Namespace: "argo", GithubToken: "ghp_test_token"})
	require.NoError(t, err)
	require.Equal(t, []string{"resolve", "write-github", "write-opencode"}, order)
	require.Equal(t, K8sContext{Name: "staging", Namespace: "argo"}, githubCtx)
	require.Equal(t, K8sContext{Name: "staging", Namespace: "argo"}, opencodeCtx)
}

func TestRunGitHubCredentialFailureHaltsPreparation(t *testing.T) {
	tests := []struct {
		name  string
		gh    GitHubCredentialsClient
		flags Flags
	}{
		{
			name:  "validation failure skips the OpenCode step",
			gh:    github.thatFailsValidation(),
			flags: flags.withKey(),
		},
		{
			name:  "configure failure skips the OpenCode step",
			gh:    github.thatFailsConfigure(),
			flags: flags.withKey(),
		},
		{
			name:  "token configure failure skips the OpenCode step",
			gh:    github.thatFailsConfigureToken(),
			flags: flags.withToken(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := setup.withMocks(
				setup.withGitHub(tt.gh),
			)
			err := cmd.Run(tt.flags)
			require.Error(t, err)
			require.False(t, opencode.configureCalled())
		})
	}
}

func TestRunSecretExistsFailureHaltsPreparation(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.thatFailsSecretExists()),
	)
	err := cmd.Run(flags.withoutKey())
	require.Error(t, err)
	require.False(t, opencode.configureCalled())
}
