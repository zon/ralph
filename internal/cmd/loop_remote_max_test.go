package cmd

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/orchestration/loop"
	"github.com/zon/ralph/internal/output"
)

// mockLoopBranchSyncClient reports the current branch in sync with the remote
// so the remote runner proceeds to submission without touching git.
type mockLoopBranchSyncClient struct{}

func (mockLoopBranchSyncClient) CurrentBranch() (string, error) { return "main", nil }
func (mockLoopBranchSyncClient) IsBranchSyncedWithRemote(string) error {
	return nil
}

// mockLoopNotifyClient swallows desktop notifications so tests never touch the
// real notifier.
type mockLoopNotifyClient struct{}

func (mockLoopNotifyClient) Error(string)   {}
func (mockLoopNotifyClient) Success(string) {}

// newRemoteLoopRunnerForTest wires the real remote loop execution path: the
// orchestration remote runner, the real loop workflow client adapter, and a
// mock argo client that captures the submitted workflow YAML. Git discovery and
// the branch-sync check are faked, so no external tool is invoked.
func newRemoteLoopRunnerForTest(t *testing.T, submittedYAML *string) loop.RemoteRunnerClient {
	t.Helper()
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			*submittedYAML = workflowYAML
			return "loop-workflow", nil
		},
	}
	adapterCtx := execcontext.NewContext()
	adapterCtx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	adapterCtx.SetRepoOwner("owner")
	adapterCtx.SetRepoName("repo")
	adapter := &loopWorkflowClientAdapter{ctx: adapterCtx, argoClient: argoClient, currentBranch: func() (string, error) { return "main", nil }}
	return loop.NewRemoteRunner(mockLoopBranchSyncClient{}, adapter, mockLoopNotifyClient{})
}

// submittedLoopContainerArgs parses a submitted loop workflow YAML and returns
// the container args of the ralph-executor template.
func submittedLoopContainerArgs(t *testing.T, workflowYAML string) []interface{} {
	t.Helper()
	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse submitted workflow YAML")
	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	args, ok := container["args"].([]interface{})
	require.True(t, ok, "the ralph-executor container has args")
	return args
}

// submittedArgValue returns the string value following the named argument in
// the container args, or "" when the args contain no such argument.
func submittedArgValue(args []interface{}, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			if v, ok := args[i+1].(string); ok {
				return v
			}
		}
	}
	return ""
}

// TestLoopRunRemoteSubmitsWorkflowWithResolvedIterationCap asserts `ralph loop
// <slug> --mode remote` submits a workflow whose container runs the loop with
// the same resolved iteration cap local mode would use: the --max flag when
// passed, otherwise the matching loop config entry's max field, otherwise the
// default of 20. The workflow YAML carries the resolved cap as an explicit
// --max argument, so the container never re-resolves it from the repository
// config.
func TestLoopRunRemoteSubmitsWorkflowWithResolvedIterationCap(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		maxFlag *int
		wantMax string
	}{
		{
			name: "config entry max caps the container loop when no flag is passed",
			config: `loops:
  - slug: fmt
    steps:
      - run gofmt
    max: 30
`,
			wantMax: "30",
		},
		{
			name: "--max caps the container loop ahead of a higher config entry max",
			config: `loops:
  - slug: fmt
    steps:
      - run gofmt
    max: 30
`,
			maxFlag: intPtr(5),
			wantMax: "5",
		},
		{
			name: "default of 20 caps the container loop when nothing sets a max",
			config: `loops:
  - slug: fmt
    steps:
      - run gofmt
`,
			wantMax: "20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeLoopConfig(t, tt.config)

			var submittedYAML string
			cmd := &LoopCmd{Mode: config.ModeRemote, Slug: "fmt", remoteRunner: newRemoteLoopRunnerForTest(t, &submittedYAML)}
			if tt.maxFlag != nil {
				cmd.Max = tt.maxFlag
			}

			err := cmd.Run()
			require.NoError(t, err)
			require.NotEmpty(t, submittedYAML, "a loop workflow is submitted in remote mode")

			args := submittedLoopContainerArgs(t, submittedYAML)
			require.Equal(t, "workflow", args[0], "the container invokes the workflow command")
			require.Equal(t, "loop", args[1], "the container invokes the workflow loop command")
			require.Equal(t, tt.wantMax, submittedArgValue(args, "--max"), "the container runs the loop with the resolved iteration cap")
		})
	}
}
