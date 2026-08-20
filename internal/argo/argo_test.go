package argo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	assert.NotNil(t, c)

	// Verify interface compliance at compile time
	var _ Client = c
}

func TestExtractWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "parses workflow name from Name field",
			output:   "Name: ralph-test-abc123\nNamespace: default\nStatus: Succeeded",
			expected: "ralph-test-abc123",
		},
		{
			name:     "returns empty string when Name field not present",
			output:   "Namespace: default\nStatus: Succeeded\nWorkflow submitted successfully",
			expected: "",
		},
		{
			name:     "handles multi-line output and extracts from correct line",
			output:   "Workflow submitted successfully\nName: ralph-feature-xyz789\nNamespace: default\nStatus: Running",
			expected: "ralph-feature-xyz789",
		},
		{
			name:     "handles Name field with extra whitespace",
			output:   "Name:    ralph-test-spaces\nStatus: Succeeded",
			expected: "ralph-test-spaces",
		},
		{
			name:     "returns empty string for empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorkflowName(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseWorkflowNames(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "header and rows returns names in order",
			output:   "NAME        STATUS   AGE\nralph-b     Running  1m\nralph-a     Succeeded 2m",
			expected: []string{"ralph-b", "ralph-a"},
		},
		{
			name:     "header only returns empty",
			output:   "NAME        STATUS   AGE",
			expected: nil,
		},
		{
			name:     "empty output returns empty",
			output:   "",
			expected: nil,
		},
		{
			name:     "rows with extra whitespace",
			output:   "NAME        STATUS   AGE\n   ralph-b     Running  1m\n\tralph-a    Succeeded 2m",
			expected: []string{"ralph-b", "ralph-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorkflowNames(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListWorkflowNames(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "argo-args.txt")
	script := fmt.Sprintf("#!/bin/bash\necho \"$@\" > %s\necho 'NAME        STATUS   AGE'\necho 'ralph-b     Running  1m'\necho 'ralph-a     Succeeded 2m'\n", argsFile)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte(script), 0755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	c := NewClient()
	names, err := c.ListWorkflowNames(K8sContext{Name: "prod-cluster", Namespace: "staging"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ralph-b", "ralph-a"}, names)

	args := readArgsFile(t, argsFile)
	assert.Contains(t, args, "-n staging")
	assert.Contains(t, args, "-l app.kubernetes.io/managed-by=ralph")
	assert.Contains(t, args, "--context prod-cluster")
}

func TestListWorkflowNamesEmpty(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "argo-args.txt")
	script := fmt.Sprintf("#!/bin/bash\necho \"$@\" > %s\necho 'NAME        STATUS   AGE'\n", argsFile)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte(script), 0755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	c := NewClient()
	names, err := c.ListWorkflowNames(K8sContext{Namespace: "staging"})
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestListWorkflowsPrintsRawOutput(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/bash\necho 'NAME        STATUS   AGE'\necho 'ralph-b     Running  1m'\necho 'ralph-a     Succeeded 2m'\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte(script), 0755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		r.Close()
		w.Close()
	})

	c := NewClient()
	require.NoError(t, c.ListWorkflows(K8sContext{Namespace: "staging"}))

	require.NoError(t, w.Close())
	output, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Contains(t, string(output), "ralph-b")
	assert.Contains(t, string(output), "ralph-a")
}

func TestListWorkflowNamesFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte("#!/bin/bash\nexit 1\n"), 0755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	c := NewClient()
	_, err := c.ListWorkflowNames(K8sContext{Namespace: "staging"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list workflows")
}

func TestLogsArgs(t *testing.T) {
	tests := []struct {
		name        string
		follow      bool
		context     string
		namespace   string
		workflow    string
		wantFollow  bool
		wantContext string
	}{
		{
			name:      "without follow",
			namespace: "staging",
			workflow:  "my-workflow",
		},
		{
			name:       "with follow",
			follow:     true,
			namespace:  "staging",
			workflow:   "my-workflow",
			wantFollow: true,
		},
		{
			name:        "with non-empty context",
			namespace:   "staging",
			workflow:    "my-workflow",
			context:     "prod-cluster",
			wantContext: "prod-cluster",
		},
		{
			name:      "with empty context omits context flag",
			namespace: "staging",
			workflow:  "my-workflow",
		},
		{
			name:        "with follow and non-empty context",
			follow:      true,
			namespace:   "staging",
			workflow:    "my-workflow",
			context:     "prod-cluster",
			wantFollow:  true,
			wantContext: "prod-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsFile := filepath.Join(t.TempDir(), "argo-args.txt")
			writeArgsRecordingArgo(t, argsFile)

			c := NewClient()
			ctx := K8sContext{Name: tt.context, Namespace: tt.namespace}
			err := c.Logs(ctx, tt.workflow, tt.follow)
			require.NoError(t, err)

			args := readArgsFile(t, argsFile)
			assert.Contains(t, args, "logs")
			assert.Contains(t, args, "-n "+tt.namespace)
			assert.Contains(t, args, tt.workflow)
			if tt.wantFollow {
				assert.Contains(t, args, "-f")
			} else {
				assert.NotContains(t, args, "-f")
			}
			if tt.wantContext != "" {
				assert.Contains(t, args, "--context "+tt.wantContext)
			} else {
				assert.NotContains(t, args, "--context")
			}
		})
	}
}

func TestLogsFailureNamesWorkflow(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte("#!/bin/bash\nexit 1\n"), 0755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	c := NewClient()
	err := c.Logs(K8sContext{Namespace: "staging"}, "ralph-test-abc123", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ralph-test-abc123")
}

func writeArgsRecordingArgo(t *testing.T, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/bash\necho \"$@\" > %s\n", argsFile)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "argo"), []byte(script), 0755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

func readArgsFile(t *testing.T, argsFile string) string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	return string(data)
}
