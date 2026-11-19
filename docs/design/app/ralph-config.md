# Config Design

Ralph reads `.ralph/config.yaml` from the repository, searching upward from the working directory. The file is optional and every field has a default, so a project runs with no configuration.

## Schema

```yaml
mode: local | worktree | remote   # default: local
items: jq query                   # default: .
cleanup: bool                     # default: true
extraIterations: int              # default: 30% of the item count, rounded up
defaultBranch: string             # default: main
model: provider/model
variant: string
agent: string

before:
  - name: string
    command: string
    args: [string]
    workDir: string
    optional: bool

services:
  - name: string
    command: string
    args: [string]
    port: int
    timeout: int                  # default: 30 seconds
    workDir: string

loops:
  - slug: string
    steps: [string]
    max: int

prompts:
  <event>: string | {file: string}   # lower camel case, e.g. pullRequest
  schedule: string | {file: string}

validate:
  model: provider/model

workflow:
  image:
    repository: string            # default: ghcr.io/zon/ralph
    tag: string                   # default: the Ralph version
  context: string
  namespace: string
  opencodeSecret: string          # default: opencode-credentials
  configMaps: [...]
  secrets: [...]
  env: {...}
  labels: {...}
  resources: {...}
```

`ralph help config` documents the nested workflow fields: mounts, environment variables, labels, and resources.

## Resolution

A command-line flag wins over the config field. The config field wins over the default. The affected options are `--mode`, `--items`, `--model`, `--variant`, `--agent`, `--context`, `--namespace`, and `--extra`. `--base` overrides `defaultBranch` for one command.

Model fields describe one model. `variant` requires a `model`, because OpenCode encodes the variant in the model reference as `provider/model#variant`. A variant with no model fails.

See [AI Design](../ai.md) for model and agent defaults.

## Prompts

`prompts` supplies the prompt for each event a [launcher](../launcher.md) handles. Keys use lower camel case for a GitHub event, or `schedule` for a scheduled run, so the `pull_request` event uses `pullRequest` and `issue_comment` uses `issueComment`. The launcher normalizes the delivered event name for lookup. See [Launcher Design](../launcher.md#prompts). Each value is a prompt string or a mapping with a `file` path relative to the repository root:

```yaml
prompts:
  push: |
    Review the repository and make one small improvement.
  pullRequest:
    file: .ralph/prompts/pull-request.md
  schedule:
    file: .ralph/prompts/daily.md
```

The launcher reads the config after it clones and fails when its event has no prompt, the prompt is empty, or the file is missing or empty. Other commands ignore `prompts`.

## Workflow Config

The `workflow` section describes remote execution. It names the container image, the Kubernetes context and namespace, the credentials the container mounts, and optional ConfigMaps, Secrets, environment variables, labels, and resources. See [Remote Mode Design](../remote.md).

## Custom Instructions

`.ralph/instructions.md` replaces the development steps in the prompt. The rest of the prompt still carries the selected item, the project file path, git history, and the report contract. The completion trailer instruction sits outside the steps, so custom instructions cannot drop it. See [Run Design](../run.md).

A [direct prompt](../prompt.md) bypasses this. See [Prompt Design](../prompt.md).
