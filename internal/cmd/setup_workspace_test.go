package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/orchestration/workspace"
)

// ---------------------------------------------------------------------------
// Mocks for orchestration WorkspaceSetup tests
// ---------------------------------------------------------------------------

type mockSWGitHubClient struct {
	configureAuthFn func(repo string) error
}

func (m *mockSWGitHubClient) ConfigureAuth(repo string) error {
	if m.configureAuthFn != nil {
		return m.configureAuthFn(repo)
	}
	return nil
}

type mockSWWorkspaceClient struct {
	setupCredentialsFn func() error
	setupSymlinksFn    func() error
}

func (m *mockSWWorkspaceClient) SetupCredentials() error {
	if m.setupCredentialsFn != nil {
		return m.setupCredentialsFn()
	}
	return nil
}

func (m *mockSWWorkspaceClient) SetupSymlinks() error {
	if m.setupSymlinksFn != nil {
		return m.setupSymlinksFn()
	}
	return nil
}

type mockSWGitClient struct {
	configureUserFn        func(name, email string)
	cloneFn                func(branch string) error
	remoteBranchExistsFn   func(branch string) (bool, error)
	fetchAndCheckoutFn     func(branch string) error
	createAndCheckoutFn    func(branch string) error
}

func (m *mockSWGitClient) ConfigureUser(name, email string) {
	if m.configureUserFn != nil {
		m.configureUserFn(name, email)
	}
}

func (m *mockSWGitClient) Clone(branch string) error {
	if m.cloneFn != nil {
		return m.cloneFn(branch)
	}
	return nil
}

func (m *mockSWGitClient) RemoteBranchExists(branch string) (bool, error) {
	if m.remoteBranchExistsFn != nil {
		return m.remoteBranchExistsFn(branch)
	}
	return false, nil
}

func (m *mockSWGitClient) FetchAndCheckout(branch string) error {
	if m.fetchAndCheckoutFn != nil {
		return m.fetchAndCheckoutFn(branch)
	}
	return nil
}

func (m *mockSWGitClient) CreateAndCheckout(branch string) error {
	if m.createAndCheckoutFn != nil {
		return m.createAndCheckoutFn(branch)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestWorkspaceSetup_HappyPath(t *testing.T) {
	t.Parallel()

	var capturedRepo string
	var capturedBotName, capturedBotEmail string

	ws := workspace.New(
		&mockSWGitHubClient{
			configureAuthFn: func(repo string) error {
				capturedRepo = repo
				return nil
			},
		},
		&mockSWWorkspaceClient{
			setupCredentialsFn: func() error { return nil },
		},
		&mockSWGitClient{
			configureUserFn: func(name, email string) {
				capturedBotName = name
				capturedBotEmail = email
			},
			cloneFn: func(branch string) error { return nil },
		},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		BotName:     "ralph-bot",
		BotEmail:    "bot@ralph.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", capturedRepo)
	assert.Equal(t, "ralph-bot", capturedBotName)
	assert.Equal(t, "bot@ralph.com", capturedBotEmail)
}

func TestWorkspaceSetup_GitHubAuthError(t *testing.T) {
	t.Parallel()

	ws := workspace.New(
		&mockSWGitHubClient{
			configureAuthFn: func(repo string) error {
				return assert.AnError
			},
		},
		&mockSWWorkspaceClient{},
		&mockSWGitClient{},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo: "owner/repo",
	})
	require.Error(t, err)
}

func TestWorkspaceSetup_SetupCredentialsError(t *testing.T) {
	t.Parallel()

	ws := workspace.New(
		&mockSWGitHubClient{},
		&mockSWWorkspaceClient{
			setupCredentialsFn: func() error {
				return assert.AnError
			},
		},
		&mockSWGitClient{},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo: "owner/repo",
	})
	require.Error(t, err)
}

func TestWorkspaceSetup_CloneError(t *testing.T) {
	t.Parallel()

	ws := workspace.New(
		&mockSWGitHubClient{},
		&mockSWWorkspaceClient{},
		&mockSWGitClient{
			cloneFn: func(branch string) error {
				return assert.AnError
			},
		},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
	})
	require.Error(t, err)
}

func TestWorkspaceSetup_TargetBranchExists(t *testing.T) {
	t.Parallel()

	var fetchedBranch string

	ws := workspace.New(
		&mockSWGitHubClient{},
		&mockSWWorkspaceClient{},
		&mockSWGitClient{
			remoteBranchExistsFn: func(branch string) (bool, error) { return true, nil },
			fetchAndCheckoutFn: func(branch string) error {
				fetchedBranch = branch
				return nil
			},
		},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo:         "owner/repo",
		TargetBranch: "feature-branch",
	})
	require.NoError(t, err)
	assert.Equal(t, "feature-branch", fetchedBranch)
}

func TestWorkspaceSetup_TargetBranchNotFound(t *testing.T) {
	t.Parallel()

	ws := workspace.New(
		&mockSWGitHubClient{},
		&mockSWWorkspaceClient{},
		&mockSWGitClient{
			remoteBranchExistsFn: func(branch string) (bool, error) { return false, nil },
		},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo:         "owner/repo",
		TargetBranch: "nonexistent",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, workspace.ErrBranchNotFound)
}

func TestWorkspaceSetup_CreateBranch(t *testing.T) {
	t.Parallel()

	var createdBranch string

	ws := workspace.New(
		&mockSWGitHubClient{},
		&mockSWWorkspaceClient{},
		&mockSWGitClient{
			remoteBranchExistsFn: func(branch string) (bool, error) { return false, nil },
			createAndCheckoutFn: func(branch string) error {
				createdBranch = branch
				return nil
			},
		},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo:         "owner/repo",
		TargetBranch: "new-branch",
		CreateBranch: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "new-branch", createdBranch)
}

func TestWorkspaceSetup_Symlinks(t *testing.T) {
	t.Parallel()

	var symlinksCalled bool

	ws := workspace.New(
		&mockSWGitHubClient{},
		&mockSWWorkspaceClient{
			setupSymlinksFn: func() error {
				symlinksCalled = true
				return nil
			},
		},
		&mockSWGitClient{},
	)

	err := ws.Setup(workspace.WorkspaceFlags{
		Repo:     "owner/repo",
		Symlinks: true,
	})
	require.NoError(t, err)
	assert.True(t, symlinksCalled)
}
