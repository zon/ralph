package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedWorkflow string
		expectedFollow   bool
		expectedCtx      string
		expectedNS       string
		wantParseErr     bool
	}{
		{
			name:             "logs command without args",
			args:             []string{"logs"},
			expectedWorkflow: "",
			expectedFollow:   false,
		},
		{
			name:             "logs command with workflow name",
			args:             []string{"logs", "my-workflow"},
			expectedWorkflow: "my-workflow",
		},
		{
			name:             "logs command with short follow flag",
			args:             []string{"logs", "-f", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedFollow:   true,
		},
		{
			name:             "logs command with follow flag",
			args:             []string{"logs", "--follow", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedFollow:   true,
		},
		{
			name:             "logs command with namespace short flag",
			args:             []string{"logs", "-n", "staging", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedNS:       "staging",
		},
		{
			name:             "logs command with namespace flag",
			args:             []string{"logs", "--namespace", "staging", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedNS:       "staging",
		},
		{
			name:             "logs command with context flag",
			args:             []string{"logs", "--context", "prod-cluster", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedCtx:      "prod-cluster",
		},
		{
			name:             "logs command with combined flags",
			args:             []string{"logs", "--context", "prod-cluster", "-n", "staging", "-f", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedFollow:   true,
			expectedCtx:      "prod-cluster",
			expectedNS:       "staging",
		},
		{
			name:         "logs command with unknown flag should fail",
			args:         []string{"logs", "--unknown-flag", "my-workflow"},
			wantParseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if tt.wantParseErr {
				if err == nil {
					t.Error("expected parse error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Logs.WorkflowName != tt.expectedWorkflow {
				t.Errorf("expected WorkflowName=%q, got %q", tt.expectedWorkflow, cmd.Logs.WorkflowName)
			}
			if cmd.Logs.Follow != tt.expectedFollow {
				t.Errorf("expected Follow=%v, got %v", tt.expectedFollow, cmd.Logs.Follow)
			}
			if cmd.Logs.Context != tt.expectedCtx {
				t.Errorf("expected Context=%q, got %q", tt.expectedCtx, cmd.Logs.Context)
			}
			if cmd.Logs.Namespace != tt.expectedNS {
				t.Errorf("expected Namespace=%q, got %q", tt.expectedNS, cmd.Logs.Namespace)
			}
		})
	}
}

// logsMock holds the paths of the mock argo script's args recording files.
type logsMock struct {
	listArgsFile string
	logsArgsFile string
}

// setupLogsMock sets up a temp working directory with a mock kubectl, a
// kubeconfig, an optional .ralph/config.yaml, and a mock argo script that
// records its arguments per subcommand (list vs logs) and returns
// logsExitCode for the logs subcommand. The mock kubectl reports no current
// context so that only explicitly passed context flags reach the argo args.
func setupLogsMock(t *testing.T, configContent, listOutput string, logsExitCode int) *logsMock {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)

	if configContent != "" {
		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644))
	}

	kubeconfigPath := filepath.Join(dir, "kubeconfig")
	kubeconfigContent := `apiVersion: v1
kind: Config
`
	require.NoError(t, os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644))
	t.Setenv("KUBECONFIG", kubeconfigPath)

	mockKubectlPath := filepath.Join(dir, "kubectl")
	mockScript := `#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo ""
elif [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo ""
fi
`
	require.NoError(t, os.WriteFile(mockKubectlPath, []byte(mockScript), 0755))
	t.Setenv("PATH", filepath.Dir(mockKubectlPath)+":"+os.Getenv("PATH"))

	listArgsFile := filepath.Join(dir, "argo-list-args.txt")
	logsArgsFile := filepath.Join(dir, "argo-logs-args.txt")
	argoScript := fmt.Sprintf(`#!/bin/bash
if [ "$1" = "list" ]; then
  echo "$@" > %s
%s
elif [ "$1" = "logs" ]; then
  echo "$@" > %s
  exit %d
fi
`, listArgsFile, listOutput, logsArgsFile, logsExitCode)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte(argoScript), 0755))

	return &logsMock{listArgsFile: listArgsFile, logsArgsFile: logsArgsFile}
}

func TestLogsCmdScenarios(t *testing.T) {
	t.Run("logs with explicit workflow name uses config namespace", func(t *testing.T) {
		mock := setupLogsMock(t, "workflow:\n  namespace: platform\n", "echo 'NAME        STATUS   AGE'\n", 0)

		cmd := &LogsCmd{WorkflowName: "test-wf"}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(mock.logsArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "logs")
		assert.Contains(t, args, "-n platform")
		assert.Contains(t, args, "test-wf")
		assert.NotContains(t, args, "-f")
		assert.NotContains(t, args, "--context")
	})

	t.Run("logs with follow set records -f", func(t *testing.T) {
		mock := setupLogsMock(t, "workflow:\n  namespace: platform\n", "echo 'NAME        STATUS   AGE'\n", 0)

		cmd := &LogsCmd{WorkflowName: "test-wf", Follow: true}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(mock.logsArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "-f")
		assert.Contains(t, args, "test-wf")
		assert.NotContains(t, args, "--context")
	})

	t.Run("custom namespace flag overrides config", func(t *testing.T) {
		mock := setupLogsMock(t, "workflow:\n  namespace: default\n", "echo 'NAME        STATUS   AGE'\n", 0)

		cmd := &LogsCmd{WorkflowName: "test-wf", Namespace: "staging"}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(mock.logsArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "-n staging")
		assert.NotContains(t, args, "-n default")
		assert.Contains(t, args, "test-wf")
	})

	t.Run("custom context flag used", func(t *testing.T) {
		mock := setupLogsMock(t, "workflow:\n  namespace: platform\n", "echo 'NAME        STATUS   AGE'\n", 0)

		cmd := &LogsCmd{WorkflowName: "test-wf", Context: "prod-cluster"}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(mock.logsArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "--context prod-cluster")
		assert.Contains(t, args, "test-wf")
	})

	t.Run("logs without a name resolves the top of the list", func(t *testing.T) {
		mock := setupLogsMock(t, "workflow:\n  namespace: platform\n", "echo 'NAME        STATUS   AGE'\necho 'ralph-b     Running  1m'\necho 'ralph-a     Succeeded 2m'\n", 0)

		cmd := &LogsCmd{}
		err := cmd.Run()
		require.NoError(t, err)

		listArgsData, err := os.ReadFile(mock.listArgsFile)
		require.NoError(t, err)
		assert.Contains(t, string(listArgsData), "list")

		argsData, err := os.ReadFile(mock.logsArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "ralph-b")
	})

	t.Run("logs without a name and an empty list", func(t *testing.T) {
		mock := setupLogsMock(t, "workflow:\n  namespace: platform\n", "echo 'NAME        STATUS   AGE'\n", 0)

		cmd := &LogsCmd{}
		err := cmd.Run()
		require.ErrorContains(t, err, "no workflows found")

		_, statErr := os.Stat(mock.logsArgsFile)
		assert.True(t, os.IsNotExist(statErr), "logs args file should not exist when no workflows are found")
	})

	t.Run("workflow pod not found", func(t *testing.T) {
		setupLogsMock(t, "workflow:\n  namespace: platform\n", "echo 'NAME        STATUS   AGE'\n", 1)

		cmd := &LogsCmd{WorkflowName: "ralph-csv-export"}
		err := cmd.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ralph-csv-export")
	})
}
