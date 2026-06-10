package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureWebhookHelpOutput(args []string) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("ralph-webhook"),
		kong.Exit(func(int) {}),
	)
	if err != nil {
		os.Stdout = old
		w.Close()
		return ""
	}

	parser.Parse(args)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

func TestServeCmdHelpText(t *testing.T) {
	output := captureWebhookHelpOutput([]string{"serve", "--help"})
	assert.Contains(t, output, "Start the webhook server")
}

func TestSetConfigCmdHelpText(t *testing.T) {
	output := captureWebhookHelpOutput([]string{"set", "config", "--help"})
	assert.Contains(t, output, "Set webhook configuration and secrets")
}

func TestWebhookCommandsParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "serve", args: []string{"serve"}},
		{name: "set config", args: []string{"set", "config"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &CLI{}
			parser, err := kong.New(cli,
				kong.Name("ralph-webhook"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)
		})
	}
}
