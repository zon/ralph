package workflow

import (
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
)

type WorkflowOptions struct {
	Image       Image
	ConfigMaps  []config.ConfigMapMount
	Secrets     []config.SecretMount
	Env         map[string]string
	KubeContext string
	Namespace   string
	Labels      map[string]string
}

func workflowOptionsFromConfig(cfg *config.RalphConfig, ctx *execcontext.Context) WorkflowOptions {
	opts := WorkflowOptions{
		Image:       MakeImage(cfg.Workflow.Image.Repository, cfg.Workflow.Image.Tag),
		ConfigMaps:  cfg.Workflow.ConfigMaps,
		Secrets:     cfg.Workflow.Secrets,
		Env:         cfg.Workflow.Env,
		KubeContext: cfg.Workflow.Context,
		Namespace:   cfg.Workflow.Namespace,
		Labels:      cfg.Workflow.Labels,
	}

	if ctx != nil && ctx.KubeContext() != "" {
		opts.KubeContext = ctx.KubeContext()
	}

	return opts
}
