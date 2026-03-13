package k8s

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretNames(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "github secret name",
			constant: GitHubSecretName,
			expected: "github-credentials",
		},
		{
			name:     "opencode secret name",
			constant: OpenCodeSecretName,
			expected: "opencode-credentials",
		},
		{
			name:     "pulumi secret name",
			constant: PulumiSecretName,
			expected: "pulumi-credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant, "Secret name should match expected value")
		})
	}
}

func TestBuildSecretArgs(t *testing.T) {
	tests := []struct {
		name          string
		secretName    string
		namespace     string
		kubeContext   string
		data          map[string]string
		expectedArgs  []string
		expectContext bool
	}{
		{
			name:        "basic secret with single data entry",
			secretName:  "my-secret",
			namespace:   "my-namespace",
			kubeContext: "",
			data:        map[string]string{"key1": "value1"},
			expectedArgs: []string{
				"create", "secret", "generic", "my-secret",
				"--from-literal=key1=value1",
				"-n", "my-namespace",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: false,
		},
		{
			name:        "secret with multiple data entries",
			secretName:  "multi-secret",
			namespace:   "default",
			kubeContext: "",
			data:        map[string]string{"user": "admin", "pass": "secret123"},
			expectedArgs: []string{
				"create", "secret", "generic", "multi-secret",
				"-n", "default",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: false,
		},
		{
			name:        "secret with context",
			secretName:  "context-secret",
			namespace:   "prod",
			kubeContext: "my-cluster",
			data:        map[string]string{"token": "abc123"},
			expectedArgs: []string{
				"create", "secret", "generic", "context-secret",
				"-n", "prod",
				"--context", "my-cluster",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: true,
		},
		{
			name:        "empty namespace uses default",
			secretName:  "default-ns-secret",
			namespace:   "",
			kubeContext: "",
			data:        map[string]string{"key": "val"},
			expectedArgs: []string{
				"create", "secret", "generic", "default-ns-secret",
				"-n", "default",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildSecretArgs(tt.secretName, tt.namespace, tt.kubeContext, tt.data)

			assert.Equal(t, "create", args[0], "First arg should be 'create'")
			assert.Equal(t, "secret", args[1], "Second arg should be 'secret'")
			assert.Equal(t, "generic", args[2], "Third arg should be 'generic'")
			assert.Equal(t, tt.secretName, args[3], "Fourth arg should be secret name")

			assert.Equal(t, "--dry-run=client", args[len(args)-3], "Third to last arg should be --dry-run=client")
			assert.Equal(t, "-o", args[len(args)-2], "Second to last arg should be -o")
			assert.Equal(t, "yaml", args[len(args)-1], "Last arg should be yaml")

			nsIdx := slices.Index(args, "-n")
			assert.NotEqual(t, -1, nsIdx, "Should have -n flag in args")
			if tt.namespace == "" {
				assert.Equal(t, "default", args[nsIdx+1], "Empty namespace should default to 'default'")
			} else {
				assert.Equal(t, tt.namespace, args[nsIdx+1], "Namespace should match")
			}

			for key, value := range tt.data {
				expectedFlag := "--from-literal=" + key + "=" + value
				assert.True(t, slices.Contains(args, expectedFlag), "Should contain %s in args", expectedFlag)
			}

			if tt.expectContext {
				assert.True(t, slices.Contains(args, "--context"), "Should contain --context flag")
				ctxIdx := slices.Index(args, "--context")
				assert.Equal(t, tt.kubeContext, args[ctxIdx+1], "Context value should match")
			} else {
				assert.False(t, slices.Contains(args, "--context"), "Should not contain --context flag")
			}
		})
	}
}

func TestBuildSecretApplyArgs(t *testing.T) {
	tests := []struct {
		name          string
		kubeContext   string
		expectContext bool
		expectedArgs  []string
	}{
		{
			name:          "apply without context",
			kubeContext:   "",
			expectContext: false,
			expectedArgs:  []string{"apply", "-f", "-"},
		},
		{
			name:          "apply with context",
			kubeContext:   "my-cluster",
			expectContext: true,
			expectedArgs:  []string{"apply", "-f", "-", "--context", "my-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildSecretApplyArgs(tt.kubeContext)

			assert.Len(t, args, len(tt.expectedArgs), "Args length should match")
			for i, expected := range tt.expectedArgs {
				assert.Equal(t, expected, args[i], "Arg %d should match", i)
			}

			if tt.expectContext {
				assert.True(t, slices.Contains(args, "--context"), "Should contain --context flag")
			}
		})
	}
}

func TestBuildConfigMapArgs(t *testing.T) {
	tests := []struct {
		name          string
		configMapName string
		namespace     string
		kubeContext   string
		data          map[string]string
		expectContext bool
	}{
		{
			name:          "basic configmap with single data entry",
			configMapName: "my-configmap",
			namespace:     "my-namespace",
			kubeContext:   "",
			data:          map[string]string{"key1": "value1"},
			expectContext: false,
		},
		{
			name:          "configmap with multiple data entries",
			configMapName: "multi-configmap",
			namespace:     "default",
			kubeContext:   "",
			data:          map[string]string{"config1": "val1", "config2": "val2"},
			expectContext: false,
		},
		{
			name:          "configmap with context",
			configMapName: "context-configmap",
			namespace:     "prod",
			kubeContext:   "my-cluster",
			data:          map[string]string{"setting": "value"},
			expectContext: true,
		},
		{
			name:          "empty namespace uses default",
			configMapName: "default-ns-configmap",
			namespace:     "",
			kubeContext:   "",
			data:          map[string]string{"key": "val"},
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildConfigMapArgs(tt.configMapName, tt.namespace, tt.kubeContext, tt.data)

			assert.Equal(t, "create", args[0], "First arg should be 'create'")
			assert.Equal(t, "configmap", args[1], "Second arg should be 'configmap'")
			assert.Equal(t, tt.configMapName, args[2], "Third arg should be configmap name")

			assert.Equal(t, "--dry-run=client", args[len(args)-3], "Third to last arg should be --dry-run=client")
			assert.Equal(t, "-o", args[len(args)-2], "Second to last arg should be -o")
			assert.Equal(t, "yaml", args[len(args)-1], "Last arg should be yaml")

			nsIdx := slices.Index(args, "-n")
			assert.NotEqual(t, -1, nsIdx, "Should have -n flag in args")
			expectedNS := tt.namespace
			if expectedNS == "" {
				expectedNS = "default"
			}
			assert.Equal(t, expectedNS, args[nsIdx+1], "Namespace should match")

			for key, value := range tt.data {
				expectedFlag := "--from-literal=" + key + "=" + value
				assert.True(t, slices.Contains(args, expectedFlag), "Should contain %s in args", expectedFlag)
			}

			if tt.expectContext {
				assert.True(t, slices.Contains(args, "--context"), "Should contain --context flag")
				ctxIdx := slices.Index(args, "--context")
				assert.Equal(t, tt.kubeContext, args[ctxIdx+1], "Context value should match")
			} else {
				assert.False(t, slices.Contains(args, "--context"), "Should not contain --context flag")
			}
		})
	}
}

func TestBuildConfigMapApplyArgs(t *testing.T) {
	tests := []struct {
		name          string
		kubeContext   string
		expectContext bool
		expectedArgs  []string
	}{
		{
			name:          "apply without context",
			kubeContext:   "",
			expectContext: false,
			expectedArgs:  []string{"apply", "-f", "-"},
		},
		{
			name:          "apply with context",
			kubeContext:   "my-cluster",
			expectContext: true,
			expectedArgs:  []string{"apply", "-f", "-", "--context", "my-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildConfigMapApplyArgs(tt.kubeContext)

			assert.Len(t, args, len(tt.expectedArgs), "Args length should match")
			for i, expected := range tt.expectedArgs {
				assert.Equal(t, expected, args[i], "Arg %d should match", i)
			}

			if tt.expectContext {
				assert.True(t, slices.Contains(args, "--context"), "Should contain --context flag")
			}
		})
	}
}
