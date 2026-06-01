package cmd

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/workspace"
)

type CommandCmd struct {
	WorkingDir string
	Command    []string
	NoNotify   bool
	NoServices bool
	Verbose    bool
	Local      bool
	Follow     bool
	Debug      string
	Context    string

	cleanupRegistrar func(func())
}

func (c *CommandCmd) Run() error {
	if err := c.changeWorkingDirectory(); err != nil {
		return err
	}

	if err := c.validateArgs(); err != nil {
		return err
	}

	flags := CommandFlags{Follow: c.Follow, Local: c.Local, Debug: c.Debug}
	if err := flags.Validate(); err != nil {
		return err
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	ctx := c.createExecutionContext()

	setup := &CommandSetup{
		Command: c.Command,
		Config:  ralphConfig,
	}

	return ExecuteCommand(ctx, c.cleanupRegistrar, setup)
}

func (c *CommandCmd) changeWorkingDirectory() error {
	if c.WorkingDir == "" {
		return nil
	}
	return workspace.Chdir(c.WorkingDir)
}

func (c *CommandCmd) validateArgs() error {
	if len(c.Command) == 0 {
		return fmt.Errorf("command required (use: ralph command -- <command> [args...]")
	}
	return nil
}

func (c *CommandCmd) createExecutionContext() *execcontext.Context {
	ctx := createExecutionContext()
	ctx.SetVerbose(c.Verbose)
	ctx.SetNoNotify(c.NoNotify)
	ctx.SetNoServices(c.NoServices)
	ctx.SetLocal(c.Local)
	ctx.SetFollow(c.Follow)
	ctx.SetDebugBranch(c.Debug)
	ctx.SetCommand(c.Command)
	return ctx
}