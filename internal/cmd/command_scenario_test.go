package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/output"
	orchestrationCommand "github.com/zon/ralph/internal/orchestration/command"
	"github.com/zon/ralph/internal/testutil"
)

func TestExecuteCommand_LocalExecution(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{}

	setup := &CommandSetup{
		Command: []string{"echo", "local-execution-test"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithLocal(true),
		testutil.WithNoNotify(true),
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.NoError(t, err)
}

func TestExecuteCommand_BeforeCommandsSequential(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{
		Before: []config.Before{
			{
				Name:    "first",
				Command: "echo",
				Args:    []string{"first"},
				Optional: false,
			},
			{
				Name:    "second",
				Command: "echo",
				Args:    []string{"second"},
				Optional: false,
			},
		},
	}

	setup := &CommandSetup{
		Command: []string{"echo", "main-command"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoNotify(true),
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.NoError(t, err)
}

func TestExecuteCommand_BeforeCommandAbortOnNonOptional(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{
		Before: []config.Before{
			{
				Name:    "will-fail",
				Command: "false",
				Args:    []string{},
				Optional: false,
			},
		},
	}

	setup := &CommandSetup{
		Command: []string{"echo", "should-not-run"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoNotify(true),
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "before")
}

func TestExecuteCommand_OptionalBeforeCommandFailureContinues(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{
		Before: []config.Before{
			{
				Name:    "optional-fail",
				Command: "false",
				Args:    []string{},
				Optional: true,
			},
		},
	}

	setup := &CommandSetup{
		Command: []string{"echo", "should-run"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoNotify(true),
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.NoError(t, err)
}

func TestExecuteCommandRemote_RemoteSubmission(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithLocal(false),
		testutil.WithNoNotify(true),
	)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))

	mockArgo := &argo.MockClient{}
	err := orchestrationCommand.ExecuteRemoteCommand(ctx, mockArgo)
	assert.NoError(t, err)
	assert.True(t, mockArgo.SubmitYAMLCalled)
	assert.False(t, mockArgo.FollowLogsCalled)
}

func TestExecuteCommandRemote_FollowLogs(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithLocal(false),
		testutil.WithFollow(true),
		testutil.WithNoNotify(true),
	)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))

	mockArgo := &argo.MockClient{}
	err := orchestrationCommand.ExecuteRemoteCommand(ctx, mockArgo)
	assert.NoError(t, err)
	assert.True(t, mockArgo.SubmitYAMLCalled)
	assert.True(t, mockArgo.FollowLogsCalled)
}