package githubtoken

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errMockFailure = errors.New("mock failure")

type mockGitHubAuthClient struct {
	getRepoFn              func(ctx context.Context) (string, string, error)
	generateAppJWTFn       func(appID string, privateKeyPEM []byte) (string, error)
	getInstallationIDFn    func(ctx context.Context, jwtToken, owner, repo string) (int64, error)
	getInstallationTokenFn func(ctx context.Context, jwtToken string, installationID int64) (string, error)
	configureGitAuthFn     func(ctx context.Context, installationToken string) error
	authenticateGHCLIFn    func(ctx context.Context, installationToken string) error
}

func (m *mockGitHubAuthClient) GetRepo(ctx context.Context) (string, string, error) {
	if m.getRepoFn != nil {
		return m.getRepoFn(ctx)
	}
	return "test-owner", "test-repo", nil
}

func (m *mockGitHubAuthClient) GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	if m.generateAppJWTFn != nil {
		return m.generateAppJWTFn(appID, privateKeyPEM)
	}
	return "test-jwt", nil
}

func (m *mockGitHubAuthClient) GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
	if m.getInstallationIDFn != nil {
		return m.getInstallationIDFn(ctx, jwtToken, owner, repo)
	}
	return 12345, nil
}

func (m *mockGitHubAuthClient) GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	if m.getInstallationTokenFn != nil {
		return m.getInstallationTokenFn(ctx, jwtToken, installationID)
	}
	return "test-installation-token", nil
}

func (m *mockGitHubAuthClient) ConfigureGitAuth(ctx context.Context, installationToken string) error {
	if m.configureGitAuthFn != nil {
		return m.configureGitAuthFn(ctx, installationToken)
	}
	return nil
}

func (m *mockGitHubAuthClient) AuthenticateGHCLI(ctx context.Context, installationToken string) error {
	if m.authenticateGHCLIFn != nil {
		return m.authenticateGHCLIFn(ctx, installationToken)
	}
	return nil
}

type mockFsClient struct {
	readFileFn func(path string) ([]byte, error)
}

func (m *mockFsClient) ReadFile(path string) ([]byte, error) {
	if m.readFileFn != nil {
		return m.readFileFn(path)
	}
	switch path {
	case "/secrets/github/app-id":
		return []byte("12345"), nil
	case "/secrets/github/private-key":
		return []byte("-----BEGIN PRIVATE KEY-----\ndummy\n-----END PRIVATE KEY-----\n"), nil
	default:
		return nil, errors.New("file not found: " + path)
	}
}

func newCmd(ghClient GitHubAuthClient, fsClient FsClient) *Cmd {
	return New(ghClient, fsClient)
}

func defaultCmd() *Cmd {
	return newCmd(&mockGitHubAuthClient{}, &mockFsClient{})
}

func TestRun_Success(t *testing.T) {
	ctx := context.Background()
	cmd := defaultCmd()
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.NoError(t, err)
}

func TestRun_OwnerAutodetection(t *testing.T) {
	ctx := context.Background()
	cmd := defaultCmd()
	err := cmd.Run(ctx, "", "test-repo", "/secrets/github")
	require.NoError(t, err)
}

func TestRun_RepoAutodetection(t *testing.T) {
	ctx := context.Background()
	cmd := defaultCmd()
	err := cmd.Run(ctx, "test-owner", "", "/secrets/github")
	require.NoError(t, err)
}

func TestRun_BothAutodetected(t *testing.T) {
	ctx := context.Background()
	cmd := defaultCmd()
	err := cmd.Run(ctx, "", "", "/secrets/github")
	require.NoError(t, err)
}

func TestRun_RepoDetectionFailure(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		getRepoFn: func(ctx context.Context) (string, string, error) {
			return "", "", errMockFailure
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "", "", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_OwnerMissingAfterDetection(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		getRepoFn: func(ctx context.Context) (string, string, error) {
			return "", "test-repo", nil
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "", "", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository owner is required")
}

func TestRun_RepoMissingAfterDetection(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		getRepoFn: func(ctx context.Context) (string, string, error) {
			return "test-owner", "", nil
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "", "", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository name is required")
}

func TestRun_MissingAppIDFile(t *testing.T) {
	ctx := context.Background()
	fsClient := &mockFsClient{
		readFileFn: func(path string) ([]byte, error) {
			if path == "/secrets/github/app-id" {
				return nil, errors.New("file not found")
			}
			return []byte("key"), nil
		},
	}
	cmd := newCmd(&mockGitHubAuthClient{}, fsClient)
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read app ID")
}

func TestRun_EmptyAppID(t *testing.T) {
	ctx := context.Background()
	fsClient := &mockFsClient{
		readFileFn: func(path string) ([]byte, error) {
			if path == "/secrets/github/app-id" {
				return []byte(""), nil
			}
			return []byte("key"), nil
		},
	}
	cmd := newCmd(&mockGitHubAuthClient{}, fsClient)
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "app ID is empty")
}

func TestRun_MissingPrivateKeyFile(t *testing.T) {
	ctx := context.Background()
	fsClient := &mockFsClient{
		readFileFn: func(path string) ([]byte, error) {
			if path == "/secrets/github/private-key" {
				return nil, errors.New("file not found")
			}
			return []byte("12345"), nil
		},
	}
	cmd := newCmd(&mockGitHubAuthClient{}, fsClient)
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read private key")
}

func TestRun_EmptyPrivateKey(t *testing.T) {
	ctx := context.Background()
	fsClient := &mockFsClient{
		readFileFn: func(path string) ([]byte, error) {
			if path == "/secrets/github/app-id" {
				return []byte("12345"), nil
			}
			return []byte(""), nil
		},
	}
	cmd := newCmd(&mockGitHubAuthClient{}, fsClient)
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private key is empty")
}

func TestRun_JWTFailure(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		generateAppJWTFn: func(appID string, privateKeyPEM []byte) (string, error) {
			return "", errMockFailure
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_InstallationIDFailure(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		getInstallationIDFn: func(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
			return 0, errMockFailure
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_InstallationTokenFailure(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		getInstallationTokenFn: func(ctx context.Context, jwtToken string, installationID int64) (string, error) {
			return "", errMockFailure
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_GitConfigFailure(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		configureGitAuthFn: func(ctx context.Context, installationToken string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_GhAuthFailure(t *testing.T) {
	ctx := context.Background()
	ghClient := &mockGitHubAuthClient{
		authenticateGHCLIFn: func(ctx context.Context, installationToken string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(ghClient, &mockFsClient{})
	err := cmd.Run(ctx, "test-owner", "test-repo", "/secrets/github")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}
