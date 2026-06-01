package cmd

import "os"

type SystemEnvClient struct{}

func (c *SystemEnvClient) InWorkflow() bool {
	return os.Getenv("RALPH_WORKFLOW_EXECUTION") == "true"
}
