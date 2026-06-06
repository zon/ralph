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

func TestListCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedCtx  string
		expectedNS   string
		wantParseErr bool
	}{
		{
			name:        "list command without flags",
			args:        []string{"list"},
			expectedCtx: "",
			expectedNS:  "",
		},
		{
			name:        "list command with context flag",
			args:        []string{"list", "--context", "my-context"},
			expectedCtx: "my-context",
			expectedNS:  "",
		},
		{
			name:        "list command with namespace flag",
			args:        []string{"list", "--namespace", "staging"},
			expectedCtx: "",
			expectedNS:  "staging",
		},
		{
			name:        "list command with namespace short flag",
			args:        []string{"list", "-n", "staging"},
			expectedCtx: "",
			expectedNS:  "staging",
		},
		{
			name:        "list command with context and namespace flags",
			args:        []string{"list", "--context", "prod-cluster", "-n", "staging"},
			expectedCtx: "prod-cluster",
			expectedNS:  "staging",
		},
		{
			name:        "list command with context and namespace long flags",
			args:        []string{"list", "--context", "prod-cluster", "--namespace", "staging"},
			expectedCtx: "prod-cluster",
			expectedNS:  "staging",
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
			if cmd.List.Namespace != tt.expectedNS {
				t.Errorf("expected Namespace=%q, got %q", tt.expectedNS, cmd.List.Namespace)
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

		mockArgoPath := filepath.Join(dir, "argo")
		require.NoError(t, os.WriteFile(mockArgoPath, []byte("#!/bin/bash\nexit 0\n"), 0755))

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

func TestListCmdScenarios(t *testing.T) {
	t.Run("default list uses config namespace", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))
		configContent := "workflow:\n  namespace: platform\n"
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

		argoArgsFile := filepath.Join(dir, "argo-args.txt")
		mockArgoPath := filepath.Join(dir, "argo")
		argoScript := fmt.Sprintf(`#!/bin/bash
echo "$@" > %s
`, argoArgsFile)
		require.NoError(t, os.WriteFile(mockArgoPath, []byte(argoScript), 0755))

		cmd := &ListCmd{}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(argoArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "-n platform")
		assert.Contains(t, args, "-l app.kubernetes.io/managed-by=ralph")
	})

	t.Run("custom namespace flag overrides config", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))
		configContent := "workflow:\n  namespace: default\n"
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

		argoArgsFile := filepath.Join(dir, "argo-args.txt")
		mockArgoPath := filepath.Join(dir, "argo")
		argoScript := fmt.Sprintf(`#!/bin/bash
echo "$@" > %s
`, argoArgsFile)
		require.NoError(t, os.WriteFile(mockArgoPath, []byte(argoScript), 0755))

		cmd := &ListCmd{Namespace: "staging"}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(argoArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "-n staging")
		assert.NotContains(t, args, "-n default")
	})

	t.Run("custom context flag used", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))
		configContent := "workflow:\n  namespace: platform\n"
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

		argoArgsFile := filepath.Join(dir, "argo-args.txt")
		mockArgoPath := filepath.Join(dir, "argo")
		argoScript := fmt.Sprintf(`#!/bin/bash
echo "$@" > %s
`, argoArgsFile)
		require.NoError(t, os.WriteFile(mockArgoPath, []byte(argoScript), 0755))

		cmd := &ListCmd{Context: "prod-cluster"}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(argoArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "--context prod-cluster")
	})

	t.Run("custom context and namespace together", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		ralphDir := filepath.Join(dir, ".ralph")
		require.NoError(t, os.MkdirAll(ralphDir, 0755))
		configContent := "workflow:\n  namespace: default\n"
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

		argoArgsFile := filepath.Join(dir, "argo-args.txt")
		mockArgoPath := filepath.Join(dir, "argo")
		argoScript := fmt.Sprintf(`#!/bin/bash
echo "$@" > %s
`, argoArgsFile)
		require.NoError(t, os.WriteFile(mockArgoPath, []byte(argoScript), 0755))

		cmd := &ListCmd{Context: "prod-cluster", Namespace: "staging"}
		err := cmd.Run()
		require.NoError(t, err)

		argsData, err := os.ReadFile(argoArgsFile)
		require.NoError(t, err)
		args := string(argsData)
		assert.Contains(t, args, "-n staging")
		assert.Contains(t, args, "--context prod-cluster")
		assert.Contains(t, args, "-l app.kubernetes.io/managed-by=ralph")
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
