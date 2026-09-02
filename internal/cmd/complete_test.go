package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"complete", "--help"})
	assert.Contains(t, output, "List the completion hashes recorded in the commit log of this branch")
	assert.Contains(t, output, "jq query selecting the project item list (default: .)")
	assert.Contains(t, output, "Base branch bounding the commit log")
	assert.Contains(t, output, "Print the hashes as a JSON array")
}

func TestCompleteCmdParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "complete", args: []string{"complete"}},
		{name: "complete with file", args: []string{"complete", "projects/csv-export.yaml"}},
		{name: "complete with items and base", args: []string{"complete", "--items", ".requirements", "--base", "main", "p.yaml"}},
		{name: "complete short base", args: []string{"complete", "-B", "main", "p.yaml"}},
		{name: "complete with json", args: []string{"complete", "--json", "projects/csv-export.yaml"}},
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

func TestGetGroupNoLongerRegistered(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"--help"})
	assert.NotContains(t, output, "Report which items are complete and which are left", "the get command group is gone now that complete and incomplete are top level")
}
