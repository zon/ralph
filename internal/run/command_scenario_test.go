package run

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
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