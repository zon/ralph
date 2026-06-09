package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/webhook"
	"github.com/zon/ralph/internal/webhookconfig"
)

type CLI struct {
	Serve ServeCmd `cmd:"" default:"withargs" help:"Start the webhook server"`
	Set   SetCmd   `cmd:"" help:"Set webhook configuration"`
}

type ServeCmd struct {
	Config  string `help:"Path to app config YAML file" env:"WEBHOOK_CONFIG"`
	Secrets string `help:"Path to secrets YAML file" env:"WEBHOOK_SECRETS"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
}

func (c *ServeCmd) Run() error {
	out := output.NewClient(os.Stdout, os.Stderr, c.Verbose)

	cfg, err := webhookconfig.LoadConfig(c.Config, c.Secrets)
	if err != nil {
		return err
	}

	s := webhook.NewServer(cfg, out)
	out.Infof("starting ralph-webhook service on port %d", cfg.App.Port)
	return s.Run()
}

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("ralph-webhook"),
		kong.Description("GitHub webhook service for ralph"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
