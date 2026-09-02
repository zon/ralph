package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncompleteCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"incomplete", "--help"})
	assert.Contains(t, output, "List project items not complete in this branch")
	assert.NotContains(t, output, "index")
}

func TestIncompleteCmdParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "incomplete with file", args: []string{"incomplete", "p.yaml"}},
		{name: "incomplete with items and base", args: []string{"incomplete", "--items", ".requirements", "--base", "main", "p.yaml"}},
		{name: "incomplete short base", args: []string{"incomplete", "-B", "main", "p.yaml"}},
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
		})
	}
}

func TestTopLevelHelpListsCompleteAndIncomplete(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"--help"})
	assert.Contains(t, output, "List the completion hashes recorded in the commit log of this branch")
	assert.Contains(t, output, "List project items not complete in this branch")
}
