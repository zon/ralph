package cmd

import (
	orchestrationCommand "github.com/zon/ralph/internal/orchestration/command"
)

type CommandCmd struct {
	Command  []string `arg:"" name:"command" help:"Command to run" optional:""`
	NoFollow bool     `help:"Skip following workflow logs" name:"no-follow" default:"false"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (c *CommandCmd) Run() error {
	cmd := orchestrationCommand.NewCommandCmd(&commandWorkflowClient{})
	flags := orchestrationCommand.CommandFlags{
		Command:  c.Command,
		NoFollow: c.NoFollow,
	}
	return cmd.Run(flags)
}

// ---------------------------------------------------------------------------
// commandWorkflowClient implements orchestration/command.WorkflowClient
// ---------------------------------------------------------------------------

type commandWorkflowClient struct{}

func (c *commandWorkflowClient) Submit(command []string) (string, error) {
	return "", nil
}

func (c *commandWorkflowClient) StreamLogs(workflowName string) error {
	return nil
}
