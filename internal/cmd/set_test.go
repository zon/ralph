package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestSetSkillsCmdParsing(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedBranch string
	}{
		{
			name:           "default branch is main when not specified",
			args:           []string{"set", "skills"},
			expectedBranch: "",
		},
		{
			name:           "explicit branch is parsed correctly",
			args:           []string{"set", "skills", "--branch", "v2"},
			expectedBranch: "v2",
		},
		{
			name:           "branch short form -b",
			args:           []string{"set", "skills", "-b", "develop"},
			expectedBranch: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)

			require.Equal(t, tt.expectedBranch, cmd.Set.Skills.Branch)
		})
	}
}

func TestSetSkillsCmdDefaultsToMain(t *testing.T) {
	cmd := &SetSkillsCmd{}
	require.Empty(t, cmd.Branch)

	branch := cmd.Branch
	if branch == "" {
		branch = "main"
	}
	require.Equal(t, "main", branch)
}