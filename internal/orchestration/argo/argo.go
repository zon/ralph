package argo

import "errors"

type K8sContext struct {
	Name      string
	Namespace string
}

type ArgoClient interface {
	List(ctx K8sContext) error
	Stop(ctx K8sContext, workflowName string) error
	ListWorkflowNames(ctx K8sContext) ([]string, error)
	Logs(ctx K8sContext, workflowName string, follow bool) error
}

type ContextClient interface {
	Resolve(flagContext, flagNamespace string) (K8sContext, error)
}

type ArgoCmd struct {
	Argo ArgoClient
	Ctx  ContextClient
}

type ListFlags struct {
	Context   string
	Namespace string
}

type StopFlags struct {
	Context      string
	Namespace    string
	WorkflowName string
}

type LogsFlags struct {
	Context      string
	Namespace    string
	WorkflowName string
	Follow       bool
}

var ErrNoWorkflows = errors.New("no workflows found")

func (c *ArgoCmd) List(flags ListFlags) error {
	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}
	return c.Argo.List(k8sCtx)
}

func (c *ArgoCmd) Stop(flags StopFlags) error {
	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}
	return c.Argo.Stop(k8sCtx, flags.WorkflowName)
}

func (c *ArgoCmd) Logs(flags LogsFlags) error {
	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}
	workflowName, err := c.resolveLogsWorkflow(k8sCtx, flags.WorkflowName)
	if err != nil {
		return err
	}
	return c.Argo.Logs(k8sCtx, workflowName, flags.Follow)
}

func (c *ArgoCmd) resolveLogsWorkflow(ctx K8sContext, requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	names, err := c.Argo.ListWorkflowNames(ctx)
	if err != nil {
		return "", err
	}
	return topWorkflow(names)
}

func topWorkflow(names []string) (string, error) {
	if len(names) == 0 {
		return "", ErrNoWorkflows
	}
	return names[0], nil
}
