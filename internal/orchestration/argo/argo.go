package argo

type K8sContext struct {
	Name      string
	Namespace string
}

type ArgoClient interface {
	List(ctx K8sContext) error
	Stop(ctx K8sContext, workflowName string) error
}

type ContextClient interface {
	Resolve(flagContext, flagNamespace string) (K8sContext, error)
}

type ArgoCmd struct {
	argo ArgoClient
	ctx  ContextClient
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

func (c *ArgoCmd) List(flags ListFlags) error {
	k8sCtx, err := c.ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}
	return c.argo.List(k8sCtx)
}

func (c *ArgoCmd) Stop(flags StopFlags) error {
	k8sCtx, err := c.ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}
	return c.argo.Stop(k8sCtx, flags.WorkflowName)
}
