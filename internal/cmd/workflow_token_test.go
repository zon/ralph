package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowTokenCmd_FlagParsing(t *testing.T) {
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
			cmd := &WorkflowTokenCmd{}

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

func TestWorkflowTokenCmd_RegisteredViaWorkflowGroup(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"workflow", "token"})
	require.NoError(t, err)

	assert.NotNil(t, cmd.Workflow.Token)
	assert.NotNil(t, cmd.Workflow.Token.Owner)
}
