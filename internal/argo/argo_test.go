package argo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	assert.NotNil(t, c)

	// Verify interface compliance at compile time
	var _ Client = c
}

func TestClientSubmitYAML_AcceptsContext(t *testing.T) {
	c := NewClient()
	_, err := c.SubmitYAML(context.Background(), "yaml", K8sContext{Name: "ctx", Namespace: "ns"})
	// We don't expect success (argo CLI isn't available), but the signature is what we're testing
	assert.Error(t, err)
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
