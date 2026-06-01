package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemEnvClient_InWorkflow_ReturnsFalseWhenEnvNotSet(t *testing.T) {
	os.Unsetenv("RALPH_WORKFLOW_EXECUTION")
	c := &SystemEnvClient{}
	require.False(t, c.InWorkflow())
}

func TestSystemEnvClient_InWorkflow_ReturnsFalseWhenEnvSetToFalse(t *testing.T) {
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "false")
	c := &SystemEnvClient{}
	require.False(t, c.InWorkflow())
}

func TestSystemEnvClient_InWorkflow_ReturnsTrueWhenEnvSetToTrue(t *testing.T) {
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "true")
	c := &SystemEnvClient{}
	require.True(t, c.InWorkflow())
}
