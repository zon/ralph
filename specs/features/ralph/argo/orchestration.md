# Argo Orchestration

## Purpose

`ralph list` and `ralph stop`: resolve the Kubernetes context and delegate to the argo client.

## Interfaces

**Module:** `internal/orchestration/argo`

```go
type ArgoClient interface {
    List(ctx K8sContext) error
    Stop(ctx K8sContext, workflowName string) error
}

type ContextClient interface {
    Resolve(flagContext, flagNamespace string) (K8sContext, error)
}
```

## Orchestration

**Module:** `internal/orchestration/argo`

```go
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
```

### Helpers

- **`c.ctx.Resolve(flagContext, flagNamespace)`** — resolves the active Kubernetes context and namespace using flag values, ralph config, and kubectl fallback, in that priority order
- **`c.argo.List(k8sCtx)`** — calls `argo list` scoped to the resolved namespace, filtered by the `app.kubernetes.io/managed-by=ralph` label selector
- **`c.argo.Stop(k8sCtx, workflowName)`** — calls `argo stop` for the named workflow scoped to the resolved namespace

## Tests

**Module:** `internal/orchestration/argo`

```go
func TestListResolvesContextAndCallsArgo(t *testing.T) {
    cmd := argo.withMocks()
    err := cmd.List(flags.anyList())
    require.NoError(t, err)
    require.True(t, argoClient.listCalled())
}

func TestListPropagatesContextResolutionFailure(t *testing.T) {
    cmd := argo.withMocks(
        argo.withContext(ctx.thatFails()),
    )
    err := cmd.List(flags.anyList())
    require.Error(t, err)
    require.False(t, argoClient.listCalled())
}

func TestStopResolvesContextAndCallsArgo(t *testing.T) {
    cmd := argo.withMocks()
    err := cmd.Stop(flags.anyStop())
    require.NoError(t, err)
    require.True(t, argoClient.stopCalled())
}

func TestStopPropagatesContextResolutionFailure(t *testing.T) {
    cmd := argo.withMocks(
        argo.withContext(ctx.thatFails()),
    )
    err := cmd.Stop(flags.anyStop())
    require.Error(t, err)
    require.False(t, argoClient.stopCalled())
}
```

### Helpers

- **`argo.withMocks(opts...)`** — constructs an `ArgoCmd` with default mock implementations; pass option helpers to override specific clients
- **`argo.withContext(client)`** — option that sets the context client
- **`flags.anyList()`** — returns a valid `ListFlags` with a non-empty context and namespace
- **`flags.anyStop()`** — returns a valid `StopFlags` with a non-empty context, namespace, and workflow name
- **`argoClient.listCalled()`** — returns true when `List` was called during the test
- **`argoClient.stopCalled()`** — returns true when `Stop` was called during the test
- **`ctx.thatFails()`** — returns a context client whose `Resolve` returns an error
