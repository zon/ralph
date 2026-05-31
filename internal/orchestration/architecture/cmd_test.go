package architecture

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/architecture"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

// ---------------------------------------------------------------------------
// Run tests
// ---------------------------------------------------------------------------

func TestRun_Success(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return &architecture.Architecture{}, nil
		},
	}
	cmd := withMocks(withArchitecture(archClient))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
		RepoOwner:           "owner",
		RepoName:            "repo",
	}
	err := cmd.Run(flags)
	require.NoError(t, err)

	assert.Len(t, aiBuildArchitecturePromptCalls(cmd), 1)
	assert.Len(t, aiRunAgentCalls(cmd), 1) // only initial generation, empty arch validates fine
	assert.Len(t, archClientLoadCalls(cmd), 1)
	assert.Len(t, gitIsFileModifiedOrNewCalls(cmd), 1)
	assert.Len(t, gitCheckoutOrCreateBranchCalls(cmd), 1)
	assert.Len(t, gitStageFileCalls(cmd), 1)
	assert.Len(t, gitCommitAllAndPushCalls(cmd), 1)
	assert.Len(t, gitHubCreatePullRequestCalls(cmd), 1)
}

func TestRun_BuildArchitecturePromptError(t *testing.T) {
	mockAI := &mockAIClient{
		buildArchitecturePromptFunc: func(output string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build architecture prompt")
}

func TestRun_RunAgentError(t *testing.T) {
	mockAI := &mockAIClient{
		runAgentFunc: func(prompt string) error {
			if prompt == "architecture prompt" {
				return errMock
			}
			return nil
		},
	}
	cmd := withMocks(withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "architecture generation failed")
}

func TestRun_ValidateAndFixError(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return nil, errMock
		},
	}
	runAgentCount := 0
	mockAI := &mockAIClient{
		buildArchitectureFixPromptFunc: func(output string, errors []string) (string, error) {
			return "fix prompt", nil
		},
		runAgentFunc: func(prompt string) error {
			runAgentCount++
			if runAgentCount > 1 {
				return errMock
			}
			return nil
		},
	}
	cmd := withMocks(withArchitecture(archClient), withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "architecture fix attempt 1 failed")
}

func TestRun_CommitAndCreatePRError(t *testing.T) {
	mockGit := &mockGitClient{
		checkoutOrCreateBranchFunc: func(name string) error {
			return errMock
		},
	}
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return &architecture.Architecture{}, nil
		},
	}
	cmd := withMocks(withArchitecture(archClient), withGit(mockGit))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
	}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to checkout architecture branch")
}

// ---------------------------------------------------------------------------
// validateAndFix tests
// ---------------------------------------------------------------------------

func TestValidateAndFix_LoadSuccessValidationPasses(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return &architecture.Architecture{}, nil
		},
	}
	cmd := withMocks(withArchitecture(archClient))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.NoError(t, err)

	assert.Len(t, archClientLoadCalls(cmd), 1)
	assert.Len(t, aiBuildArchitectureFixPromptCalls(cmd), 0)
}

func TestValidateAndFix_LoadErrorFixSucceeds(t *testing.T) {
	loadAttempts := 0
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			loadAttempts++
			if loadAttempts == 1 {
				return nil, errMock
			}
			return &architecture.Architecture{}, nil
		},
	}
	cmd := withMocks(withArchitecture(archClient))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.NoError(t, err)

	assert.Equal(t, 2, loadAttempts)
	fixCalls := aiBuildArchitectureFixPromptCalls(cmd)
	require.Len(t, fixCalls, 1)
	assert.Equal(t, "architecture.yaml", fixCalls[0].output)
	assert.Equal(t, []string{"mock error"}, fixCalls[0].errors)
}

func TestValidateAndFix_LoadErrorMaxAttemptsExceeded(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return nil, errMock
		},
	}
	mockAI := &mockAIClient{
		buildArchitectureFixPromptFunc: func(output string, errors []string) (string, error) {
			return "fix prompt", nil
		},
	}
	cmd := withMocks(withArchitecture(archClient), withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load architecture file after 3 attempts")

	// 3 load calls; fix prompts only for attempts 1 and 2 (attempt 3 returns early)
	assert.Len(t, archClientLoadCalls(cmd), 3)
	assert.Len(t, aiBuildArchitectureFixPromptCalls(cmd), 2)
	assert.Len(t, aiRunAgentCalls(cmd), 2)
}

func TestValidateAndFix_ValidationErrorFixSucceeds(t *testing.T) {
	loadAttempts := 0
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			loadAttempts++
			if loadAttempts == 1 {
				return &architecture.Architecture{
					Modules: []architecture.Module{
						{Path: "", Description: "desc", Type: "domain"},
					},
				}, nil
			}
			return &architecture.Architecture{}, nil
		},
	}
	cmd := withMocks(withArchitecture(archClient))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.NoError(t, err)

	assert.Equal(t, 2, loadAttempts)
	fixCalls := aiBuildArchitectureFixPromptCalls(cmd)
	require.Len(t, fixCalls, 1)
	assert.Contains(t, fixCalls[0].errors[0], "path is required")
}

func TestValidateAndFix_ValidationErrorMaxAttemptsExceeded(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return &architecture.Architecture{
				Modules: []architecture.Module{
					{Path: "", Description: "desc", Type: "domain"},
				},
			}, nil
		},
	}
	mockAI := &mockAIClient{
		buildArchitectureFixPromptFunc: func(output string, errors []string) (string, error) {
			return "fix prompt", nil
		},
	}
	cmd := withMocks(withArchitecture(archClient), withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "architecture validation failed after 3 attempts")

	// 3 load calls; fix prompts only for attempts 1 and 2 (attempt 3 returns with final error)
	assert.Len(t, archClientLoadCalls(cmd), 3)
	assert.Len(t, aiBuildArchitectureFixPromptCalls(cmd), 2)
	assert.Len(t, aiRunAgentCalls(cmd), 2)
}

func TestValidateAndFix_BuildFixPromptError(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return nil, errMock
		},
	}
	mockAI := &mockAIClient{
		buildArchitectureFixPromptFunc: func(output string, errors []string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withArchitecture(archClient), withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build fix prompt")
}

func TestValidateAndFix_RunAgentFixError(t *testing.T) {
	archClient := &mockArchitectureClient{
		loadFunc: func(path string) (*architecture.Architecture, error) {
			return nil, errMock
		},
	}
	mockAI := &mockAIClient{
		buildArchitectureFixPromptFunc: func(output string, errors []string) (string, error) {
			return "fix prompt", nil
		},
		runAgentFunc: func(prompt string) error {
			return errMock
		},
	}
	cmd := withMocks(withArchitecture(archClient), withAI(mockAI))

	flags := ArchitectureFlags{Output: "architecture.yaml"}
	err := cmd.validateAndFix(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "architecture fix attempt 1 failed")
}

// ---------------------------------------------------------------------------
// commitAndCreatePR tests
// ---------------------------------------------------------------------------

func TestCommitAndCreatePR_NotWorkflowExecution(t *testing.T) {
	cmd := withMocks()

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: false,
	}
	err := cmd.commitAndCreatePR(flags)
	require.NoError(t, err)

	assert.Len(t, gitIsFileModifiedOrNewCalls(cmd), 0)
	assert.Len(t, gitCheckoutOrCreateBranchCalls(cmd), 0)
	assert.Len(t, gitStageFileCalls(cmd), 0)
	assert.Len(t, gitCommitAllAndPushCalls(cmd), 0)
	assert.Len(t, gitHubCreatePullRequestCalls(cmd), 0)
}

func TestCommitAndCreatePR_NoChangesDetected(t *testing.T) {
	mockGit := &mockGitClient{
		isFileModifiedOrNewFunc: func(path string) bool {
			return false
		},
	}
	cmd := withMocks(withGit(mockGit))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
	}
	err := cmd.commitAndCreatePR(flags)
	require.NoError(t, err)

	assert.Len(t, gitIsFileModifiedOrNewCalls(cmd), 1)
	assert.Len(t, gitCheckoutOrCreateBranchCalls(cmd), 0)
	assert.Len(t, gitStageFileCalls(cmd), 0)
	assert.Len(t, gitCommitAllAndPushCalls(cmd), 0)
	assert.Len(t, gitHubCreatePullRequestCalls(cmd), 0)
}

func TestCommitAndCreatePR_Success(t *testing.T) {
	cmd := withMocks()

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
		RepoOwner:           "owner",
		RepoName:            "repo",
		BaseBranch:          "main",
	}
	err := cmd.commitAndCreatePR(flags)
	require.NoError(t, err)

	assert.Len(t, gitIsFileModifiedOrNewCalls(cmd), 1)
	assert.Len(t, gitCheckoutOrCreateBranchCalls(cmd), 1)
	assert.Equal(t, "architecture", gitCheckoutOrCreateBranchCalls(cmd)[0])
	assert.Len(t, gitStageFileCalls(cmd), 1)
	assert.Equal(t, "architecture.yaml", gitStageFileCalls(cmd)[0])

	commitCalls := gitCommitAllAndPushCalls(cmd)
	require.Len(t, commitCalls, 1)
	assert.Equal(t, "owner", commitCalls[0].auth.Owner)
	assert.Equal(t, "repo", commitCalls[0].auth.Repo)
	assert.Equal(t, "architecture", commitCalls[0].branchName)
	assert.Equal(t, "architecture: generate architecture.yaml", commitCalls[0].commitMsg)

	prCalls := gitHubCreatePullRequestCalls(cmd)
	require.Len(t, prCalls, 1)
	assert.Equal(t, "architecture", prCalls[0].branchName)
	assert.Equal(t, "main", prCalls[0].baseBranch)
	assert.Equal(t, "Automatically generated architecture.yaml documenting the project structure.", prCalls[0].prSummary)
	assert.Equal(t, "architecture", prCalls[0].proj.Slug)
	assert.Equal(t, "Update architecture.yaml", prCalls[0].proj.Title)
}

func TestCommitAndCreatePR_DefaultBaseBranch(t *testing.T) {
	cmd := withMocks()

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
		RepoOwner:           "owner",
		RepoName:            "repo",
		BaseBranch:          "",
	}
	err := cmd.commitAndCreatePR(flags)
	require.NoError(t, err)

	prCalls := gitHubCreatePullRequestCalls(cmd)
	require.Len(t, prCalls, 1)
	assert.Equal(t, "main", prCalls[0].baseBranch)
}

func TestCommitAndCreatePR_CheckoutBranchError(t *testing.T) {
	mockGit := &mockGitClient{
		checkoutOrCreateBranchFunc: func(name string) error {
			return errMock
		},
	}
	cmd := withMocks(withGit(mockGit))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
	}
	err := cmd.commitAndCreatePR(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to checkout architecture branch")
}

func TestCommitAndCreatePR_StageFileError(t *testing.T) {
	mockGit := &mockGitClient{
		stageFileFunc: func(path string) error {
			return errMock
		},
	}
	cmd := withMocks(withGit(mockGit))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
	}
	err := cmd.commitAndCreatePR(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stage architecture file")
}

func TestCommitAndCreatePR_CommitAndPushError(t *testing.T) {
	mockGit := &mockGitClient{
		commitAllAndPushFunc: func(auth *git.AuthConfig, branchName, commitMsg string) error {
			return errMock
		},
	}
	cmd := withMocks(withGit(mockGit))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
	}
	err := cmd.commitAndCreatePR(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit and push architecture")
}

func TestCommitAndCreatePR_CreatePRError(t *testing.T) {
	mockGH := &mockGitHubClient{
		createPullRequestFunc: func(proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withGitHub(mockGH))

	flags := ArchitectureFlags{
		Output:              "architecture.yaml",
		IsWorkflowExecution: true,
	}
	err := cmd.commitAndCreatePR(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create pull request")
}
