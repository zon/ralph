# Launcher Design

`ralph-launcher` is one release per Ralph namespace. It listens to the repository subject published by [ralph-events](events.md) and starts a Ralph prompt workflow in its namespace.

The chart lives in `charts/ralph-launcher`.

## Flow

1. A NATS `EventSource` subscribes to `ralph.repo.<owner>.<name>` on the shared NATS server.
2. The event source republishes the message to the namespace event bus.
3. The sensor filters the event and starts a `Workflow`.
4. The `WorkflowTemplate` runs `ralph workflow prompt` in the Ralph image.
5. The container clones `repo.cloneBranch`, checks out `prompt.branch`, runs the prompt text, pushes the branch, and opens a pull request.

The launcher owns the namespace event bus, the event source, the sensor, the `WorkflowTemplate`, and the prompt ConfigMap. Argo Workflows owns the pods the workflow creates. The launcher assumes a NATS JetStream server already runs on the cluster and does not install NATS. Its NATS event source presents the service account token from `nats.auth.tokenSecret`. The namespace maps to a NATS account in the auth callout.

## Prompt ConfigMap

`prompt.text` is written to a ConfigMap. The workflow argument `prompt` reads that key with `valueFrom.configMapKeyRef`. A ConfigMap edit changes the prompt for the next workflow without changing the sensor or template.

The prompt branch defaults to `ralph-prompt-<namespace>` so separate namespaces do not share a branch. A workflow mutex serializes runs on that branch.

## Filters

The default dependency filters accept `push` events on `refs/heads/<repo.baseBranch>` and drop pushes from `ralph-zon[bot]`. This prevents a Ralph push from starting another prompt run.

Set `sensor.events` to the GitHub event names that should start a workflow. The default is `[push]`. An empty list accepts every event:

```yaml
sensor:
  events:
    - push
    - pull_request
```

Set `sensor.defaultBranchOnly=false` to accept pushes to other branches. Add `sensor.filters` for extra Argo Events data filters. Fields are relative to the NATS event data, so the GitHub event name sits at `body.event` and the body sits under `body.payload`.

## Workflow

The `WorkflowTemplate` passes these values to the container:

```text
ralph workflow prompt --repo <owner/name> --clone <clone-branch>
  --branch <prompt-branch> --base <base-branch> --prompt <prompt>
```

It adds `--model`, `--variant`, `--agent`, `--no-services`, and `--verbose` when the values set them.

The workflow mounts `github-credentials` at `/secrets/github` and `opencode-credentials` at `/secrets/opencode`. `ralph setup` writes both Secrets. See [Setup Design](setup.md) and [Workflow Container Design](workflow.md).

## Event Bus

Each namespace needs an Argo Events event bus. The chart creates a bus named `default` with `jetstreamExotic` pointing at the existing NATS server. It does not create a NATS StatefulSet. When a namespace already has a bus, set `eventBus.create=false` and point `eventBus.name` at it.

All launcher releases share one NATS JetStream instance. The sensor and event source names include the release namespace so their NATS subjects and consumers do not collide with another launcher. Every `EventBus` that points at the shared NATS server should use the same stream settings.

## RBAC

The sensor uses a ServiceAccount with permission to create and inspect workflows and read workflow templates. The chart creates the ServiceAccount, Role, and RoleBinding. The workflow pods use the namespace default ServiceAccount unless `workflow.serviceAccountName` names another one.
