package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCmdHelpListsCompleteAndIncomplete(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"get", "--help"})
	assert.Contains(t, output, "List the completion hashes recorded in the commit log of this branch")
	assert.Contains(t, output, "List the items that are not complete")
}

func TestGetCompleteCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"get", "complete", "--help"})
	assert.Contains(t, output, "List the completion hashes recorded in the commit log of this branch")
	assert.Contains(t, output, "jq query selecting the project item list (default: .)")
	assert.Contains(t, output, "Base branch bounding the commit log")
}

func TestGetIncompleteCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"get", "incomplete", "--help"})
	assert.Contains(t, output, "List the items that are not complete")
	assert.Contains(t, output, "Emit indices instead of items")
}

func TestGetCommandsParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "get complete", args: []string{"get", "complete"}},
		{name: "get complete with file", args: []string{"get", "complete", "projects/csv-export.yaml"}},
		{name: "get complete with items and base", args: []string{"get", "complete", "--items", ".requirements", "--base", "main", "p.yaml"}},
		{name: "get complete short base", args: []string{"get", "complete", "-B", "main", "p.yaml"}},
		{name: "get incomplete", args: []string{"get", "incomplete", "p.yaml"}},
		{name: "get incomplete with index", args: []string{"get", "incomplete", "--index", "p.yaml"}},
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
