package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedCtx  string
		wantParseErr bool
	}{
		{
			name:        "list command without flags",
			args:        []string{"list"},
			expectedCtx: "",
		},
		{
			name:        "list command with context flag",
			args:        []string{"list", "--context", "my-context"},
			expectedCtx: "my-context",
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

			if cmd.List.Context != tt.expectedCtx {
				t.Errorf("expected Context=%q, got %q", tt.expectedCtx, cmd.List.Context)
			}
		})
	}
}

func TestStopCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedWorkflow string
		expectedCtx      string
		wantParseErr     bool
	}{
		{
			name:             "stop command with workflow name",
			args:             []string{"stop", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedCtx:      "",
		},
		{
			name:             "stop command with workflow name and context flag",
			args:             []string{"stop", "--context", "my-context", "my-workflow"},
			expectedWorkflow: "my-workflow",
			expectedCtx:      "my-context",
		},
		{
			name:         "stop command without workflow name should fail",
			args:         []string{"stop"},
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

			if cmd.Stop.WorkflowName != tt.expectedWorkflow {
				t.Errorf("expected WorkflowName=%q, got %q", tt.expectedWorkflow, cmd.Stop.WorkflowName)
			}
			if cmd.Stop.Context != tt.expectedCtx {
				t.Errorf("expected Context=%q, got %q", tt.expectedCtx, cmd.Stop.Context)
			}
		})
	}
}

func TestListCmdNamespaceResolution(t *testing.T) {
	setupMocks := func(t *testing.T, dir string) {
		mockKubectlPath := filepath.Join(dir, "kubectl")
		mockScript := `#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo "test-ctx"
elif [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo "kubectl-ns"
fi
`
		require.NoError(t, os.WriteFile(mockKubectlPath, []byte(mockScript), 0755))
		t.Setenv("PATH", filepath.Dir(mockKubectlPath)+":"+os.Getenv("PATH"))
	}

	t.Run("uses namespace from config", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))

		configContent := "workflow:\n  namespace: config-ns\n"
		require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644))

		kubeconfigPath := filepath.Join(dir, "kubeconfig")
		kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-ctx
contexts:
- name: test-ctx
  context:
    namespace: kubectl-ns
`
		require.NoError(t, os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644))
		t.Setenv("KUBECONFIG", kubeconfigPath)

		setupMocks(t, dir)

		cmd := &ListCmd{}
		err := cmd.Run()
		assert.NoError(t, err)
	})

	t.Run("uses namespace from kubectl when no config", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		kubeconfigPath := filepath.Join(dir, "kubeconfig")
		kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-ctx
contexts:
- name: test-ctx
  context:
    namespace: kubectl-ns
`
		require.NoError(t, os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644))
		t.Setenv("KUBECONFIG", kubeconfigPath)

		setupMocks(t, dir)

		cmd := &ListCmd{}
		err := cmd.Run()
		assert.Error(t, err)
	})

	t.Run("defaults to default namespace", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		kubeconfigPath := filepath.Join(dir, "kubeconfig")
		kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-ctx
contexts:
- name: test-ctx
`
		require.NoError(t, os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644))
		t.Setenv("KUBECONFIG", kubeconfigPath)

		mockKubectlPath := filepath.Join(dir, "kubectl")
		mockScript := `#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo "test-ctx"
elif [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo ""
fi
`
		require.NoError(t, os.WriteFile(mockKubectlPath, []byte(mockScript), 0755))
		t.Setenv("PATH", filepath.Dir(mockKubectlPath)+":"+os.Getenv("PATH"))

		cmd := &ListCmd{}
		err := cmd.Run()
		assert.Error(t, err)
	})
}

func TestStopCmdNamespaceResolution(t *testing.T) {
	setupMocks := func(t *testing.T, dir string) {
		mockKubectlPath := filepath.Join(dir, "kubectl")
		mockScript := `#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo "test-ctx"
elif [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo "kubectl-ns"
fi
`
		require.NoError(t, os.WriteFile(mockKubectlPath, []byte(mockScript), 0755))
		t.Setenv("PATH", filepath.Dir(mockKubectlPath)+":"+os.Getenv("PATH"))
	}

	t.Run("uses namespace from config", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))

		configContent := "workflow:\n  namespace: config-ns\n"
		require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644))

		kubeconfigPath := filepath.Join(dir, "kubeconfig")
		kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-ctx
contexts:
- name: test-ctx
  context:
    namespace: kubectl-ns
`
		require.NoError(t, os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644))
		t.Setenv("KUBECONFIG", kubeconfigPath)

		setupMocks(t, dir)

		cmd := &StopCmd{WorkflowName: "test-wf"}
		err := cmd.Run()
		assert.Error(t, err)
	})

	t.Run("requires workflow name argument", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		kubeconfigPath := filepath.Join(dir, "kubeconfig")
		kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-ctx
contexts:
- name: test-ctx
`
		require.NoError(t, os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644))
		t.Setenv("KUBECONFIG", kubeconfigPath)

		mockKubectlPath := filepath.Join(dir, "kubectl")
		mockScript := `#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo "test-ctx"
elif [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo ""
fi
`
		require.NoError(t, os.WriteFile(mockKubectlPath, []byte(mockScript), 0755))
		t.Setenv("PATH", filepath.Dir(mockKubectlPath)+":"+os.Getenv("PATH"))

		cmd := &StopCmd{}
		err := cmd.Run()
		assert.Error(t, err)
	})
}
