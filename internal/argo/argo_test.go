package argo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestK8sContext(t *testing.T) {
	ctx := K8sContext{
		Name:      "test-context",
		Namespace: "test-namespace",
	}

	assert.Equal(t, "test-context", ctx.Name)
	assert.Equal(t, "test-namespace", ctx.Namespace)
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
			result := ExtractWorkflowName(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}
