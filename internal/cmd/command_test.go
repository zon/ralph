package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandCmd_MissingCommand(t *testing.T) {
	cmd := &CommandCmd{}
	err := cmd.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be empty")
}

// TestCommandCmdParsing covers the `ralph command` command surface. It checks
// the positional command tokens and the --context and --namespace flags.
func TestCommandCmdParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantCommand   []string
		wantNoFollow  bool
		wantVerbose   bool
		wantContext   string
		wantNamespace string
	}{
		{
			name:        "command tokens parse",
			args:        []string{"command", "echo", "hello"},
			wantCommand: []string{"echo", "hello"},
		},
		{
			name:          "default values are empty when not provided",
			args:          []string{"command", "echo"},
			wantCommand:   []string{"echo"},
			wantContext:   "",
			wantNamespace: "",
		},
		{
			name:          "explicit --context is parsed correctly",
			args:          []string{"command", "echo", "--context", "prod"},
			wantCommand:   []string{"echo"},
			wantContext:   "prod",
			wantNamespace: "",
		},
		{
			name:          "explicit --namespace is parsed correctly",
			args:          []string{"command", "echo", "--namespace", "argo"},
			wantCommand:   []string{"echo"},
			wantContext:   "",
			wantNamespace: "argo",
		},
		{
			name:          "explicit -n short form is parsed correctly",
			args:          []string{"command", "echo", "-n", "staging"},
			wantCommand:   []string{"echo"},
			wantContext:   "",
			wantNamespace: "staging",
		},
		{
			name:          "--context and --namespace parse alongside each other",
			args:          []string{"command", "echo", "--context", "prod", "--namespace", "argo"},
			wantCommand:   []string{"echo"},
			wantContext:   "prod",
			wantNamespace: "argo",
		},
		{
			name:         "--no-follow parses",
			args:         []string{"command", "echo", "--no-follow"},
			wantCommand:  []string{"echo"},
			wantNoFollow: true,
		},
		{
			name:        "--verbose parses",
			args:        []string{"command", "echo", "--verbose"},
			wantCommand: []string{"echo"},
			wantVerbose: true,
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
			assert.Equal(t, tt.wantCommand, cmd.Command.Command)
			assert.Equal(t, tt.wantNoFollow, cmd.Command.NoFollow)
			assert.Equal(t, tt.wantVerbose, cmd.Command.Verbose)
			assert.Equal(t, tt.wantContext, cmd.Command.Context)
			assert.Equal(t, tt.wantNamespace, cmd.Command.Namespace)
		})
	}
}

// TestCommandCmdNewExecutionContextWiresKubeTargeting asserts the --context and
// --namespace flags are carried on the execution context so the workflow
// submission and any followed logs read the overrides downstream.
func TestCommandCmdNewExecutionContextWiresKubeTargeting(t *testing.T) {
	tests := []struct {
		name          string
		command       CommandCmd
		wantContext   string
		wantNamespace string
	}{
		{
			name: "flags flow into the context",
			command: CommandCmd{
				Context:   "prod-cluster",
				Namespace: "argo",
			},
			wantContext:   "prod-cluster",
			wantNamespace: "argo",
		},
		{
			name:          "unset flags leave the context empty for config fallback",
			wantContext:   "",
			wantNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.command.newExecutionContext()
			assert.Equal(t, tt.wantContext, ctx.KubeContext(), "the kube context override is applied to the context")
			assert.Equal(t, tt.wantNamespace, ctx.KubeNamespace(), "the kube namespace override is applied to the context")
			assert.Equal(t, tt.command.Command, ctx.Command(), "the command tokens are applied to the context")
		})
	}
}

// TestCommandCmdHelpText asserts the command subcommand help lists the
// --context and --namespace options required by the kubectl targeting contract.
func TestCommandCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"command", "--help"})
	assert.Contains(t, output, "Run a command in a remote Ralph workflow")
	assert.Contains(t, output, "--context")
	assert.Contains(t, output, "--namespace")
	assert.Contains(t, output, "-n")
}
