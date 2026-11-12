package workflow

import (
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/k8s"
)

type WorkflowOptions struct {
	Image          Image
	ConfigMaps     []config.ConfigMapMount
	Secrets        []config.SecretMount
	OpenCodeSecret string
	Env            map[string]config.EnvVar
	KubeContext    string
	Namespace      string
	Labels         map[string]string
	Resources      config.WorkflowResources
}

func workflowOptionsFromConfig(cfg *config.RalphConfig, ctx *execcontext.Context) (WorkflowOptions, error) {
	opts := WorkflowOptions{
		Image:          MakeImage(cfg.Workflow.Image.Repository, cfg.Workflow.Image.Tag),
		ConfigMaps:     cfg.Workflow.ConfigMaps,
		Secrets:        cfg.Workflow.Secrets,
		OpenCodeSecret: cfg.Workflow.OpenCodeSecret,
		Env:            cfg.Workflow.Env,
		KubeContext:    cfg.Workflow.Context,
		Namespace:      cfg.Workflow.Namespace,
		Labels:         cfg.Workflow.Labels,
		Resources:      cfg.Workflow.Resources,
	}

	if k8s.InCluster() {
		opts.KubeContext = ""
		if ctx != nil && ctx.KubeNamespace() != "" {
			opts.Namespace = ctx.KubeNamespace()
		}
		if opts.Namespace == "" {
			namespace, err := k8s.PodNamespace()
			if err != nil {
				return WorkflowOptions{}, err
			}
			opts.Namespace = namespace
		}
		return opts, nil
	}

	if ctx != nil && ctx.KubeContext() != "" {
		opts.KubeContext = ctx.KubeContext()
	}

	if ctx != nil && ctx.KubeNamespace() != "" {
		opts.Namespace = ctx.KubeNamespace()
	}

	return opts, nil
}
