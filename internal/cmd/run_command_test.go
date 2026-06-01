package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/testutil"
)

func TestExecuteCommand_BeforeCommandFailure(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{
		Before: []config.Before{
			{
				Name:    "failing-before",
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
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run before commands")
}

func TestExecuteCommand_CommandFailure(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{}

	setup := &CommandSetup{
		Command: []string{"false"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoNotify(true),
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command failed")
}

func TestExecuteCommand_CommandSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{}

	setup := &CommandSetup{
		Command: []string{"echo", "hello"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoNotify(true),
	)

	err := ExecuteCommand(ctx, func(f func()) {}, setup)
	assert.NoError(t, err)
}

func TestExecuteCommandRemote_SubmitsWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := tmpDir + "/test-project.yaml"
	require.NoError(t, os.WriteFile(projectFile, []byte(`slug: test-project
title: Test project
requirements: []
`), 0644))

	ralphConfig := &config.RalphConfig{}

	setup := &CommandSetup{
		Command: []string{"echo", "hello"},
		Config:  ralphConfig,
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithLocal(false),
		testutil.WithNoNotify(true),
	)

	err := executeCommandRemote(ctx, setup)
	assert.NoError(t, err)
}