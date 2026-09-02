# Configuration

Ralph looks for `.ralph/config.yaml` in your project root for optional settings.

## Format

```yaml
mode: worktree                 # default execution mode (worktree, local, or remote; default: worktree)
items: .requirements           # jq query selecting the item array from a project file (default: .)
cleanup: false                 # Delete the project file in its own commit once complete (default: false)
extraIterations:               # Extra iterations beyond item count (unset = 20% of items, rounded up)
defaultBranch: main             # Default branch for PRs (default: main)
model: deepseek/deepseek-chat  # AI model (default: deepseek/deepseek-chat)
agent: build                   # opencode agent used for coding (optional; default: opencode's primary agent)
variant: high                  # provider-specific reasoning effort (optional)

before:
  - name: compile
    command: go
    args: [build, -o, bin/app, ./cmd/app]
    workDir: /path/to/project  # optional

services:
  - name: api-server
    command: npm
    args: [run, dev]
    port: 3000                 # optional: port for health checking

workflow:
  image:
    repository: ghcr.io/zon/ralph
    tag: latest                # optional; defaults to the current Ralph version
  context: my-cluster          # kubectl context (optional)
  namespace: argo              # workflow namespace (optional)
  configMaps:                  # additional ConfigMaps to mount (optional)
    - name: app-config         # name of the ConfigMap
  secrets:                     # additional Secrets to mount (optional)
    - name: api-keys           # name of the Secret
  env:                         # environment variables (optional)
    DEBUG: "true"
    API_KEY:                   # value from a Kubernetes secret
      secretKeyRef:
        name: my-secret
        key: api-key
  labels:                      # Kubernetes labels for workflow pods (optional)
    environment: production
    team: platform
  resources:                   # CPU and memory requests and limits (optional)
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: "1"
      memory: 1Gi
```

API keys are managed by OpenCode, not Ralph. Configure them with `opencode auth`.

## Mode

`mode` selects the default execution mode for `ralph run` and `ralph loop`. It can be one of:

| Value | Description |
|-------|-------------|
| `worktree` | Runs the loop in-process in a Git worktree created in a sibling directory, leaving the current checkout untouched (default) |
| `local` | Runs the loop in-process in the current checkout |
| `remote` | Submits an Argo Workflow to Kubernetes and runs the loop in the container |

`--mode` on the command line takes priority over this field, which takes priority over the `worktree` default.

```yaml
mode: remote
```

## Items

`items` is a [jq](https://jqlang.org/manual/) query that selects the array of work items from a project file. It defaults to `.`, which is correct for a project file whose top level is already an array.

```yaml
items: .requirements                                  # nested list
items: .spec.tasks                                    # deeper nesting
items: '.issues | map(select(.state == "open"))'      # filtered
```

The query must resolve to at least one non-empty item. Empty outputs, null, `false`, `0`, blank strings, `{}`, `[]`, are dropped before indexing. Every command that reads a project file, the run command, `ralph get`, and `ralph validate`, resolves it the same way: `--items` first, then this field, then `.`. Keep the query stable for the duration of a run. It defines the items that completion tracking hashes. See [Project Files](../../docs/projects.md#item-query) and [Iterations](../../docs/iterations.md).

## Iterations

`extraIterations` sets how many iterations the loop may run beyond the item count. The limit is `len(items) + extraIterations`. When unset it defaults to 20% of the item count, rounded up. `--extra` overrides it.

`cleanup` deletes the project file once every item is complete, in a commit of its own, before the pull request is opened. Off by default. `--cleanup` enables it for a single run. Completion history lives in the branch's commit trailers, so cleaning up the file does not lose it.

## Loops

`loops` defines named step lists for the `ralph loop` command. Each entry has a `slug` and `steps`.

```yaml
loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
  - slug: update-deps
    steps:
      - check for outdated dependencies
      - update the dependency manifest
      - run the test suite
```

`ralph loop <slug>` uses the entry whose `slug` matches and embeds its `steps` in the prompt. When no entry matches, it returns `loop config not found: <slug>`.

## Before

`before` defines commands that run once before services start and before the iteration loop begins.

- Commands run sequentially and must exit successfully before Ralph proceeds (unless marked optional)
- Each entry requires `name` and `command`. `args`, `workDir`, and `optional` are optional
- Set `optional: true` to allow a command to fail without aborting the run (a warning is logged instead)
- Useful for compilation, code generation, dependency installation, database migrations

## Services

`services` defines processes to start before the iteration loop and stop after execution.

- Services are started in order
- Health checks wait for TCP ports to respond if `port` is specified
- Services are stopped gracefully (SIGTERM) after execution
- Use `--no-services` to skip service management

## Validate

`validate` configures the AI repair prompt behind `ralph validate`. When a project file fails to parse, `ralph validate` runs a bounded fix loop using the `validate.model`, falling back to the top-level `model` when it is unset:

```yaml
validate:
  model: google/gemini-2.5-pro   # optional: AI model override (default: the root 'model')
```

## Workflow

`workflow` configures remote execution on Kubernetes via Argo Workflows. All fields are optional.

| Field | Description |
|-------|-------------|
| `image.repository` | Container image (default: `ghcr.io/zon/ralph`) |
| `image.tag` | Image tag (default: the current Ralph version) |
| `context` | kubectl context to use (optional) |
| `namespace` | Kubernetes namespace for the workflow and its credentials (optional) |
| `configMaps` | Additional ConfigMaps to mount into the container |
| `secrets` | Additional Secrets to mount into the container |
| `env` | Environment variables to set in the container. Each value is a literal string or a Kubernetes secret reference |
| `labels` | Kubernetes labels to apply to workflow pods |
| `resources` | CPU and memory requests and limits for the container |

A `configMaps` or `secrets` entry names a Kubernetes ConfigMap or Secret to mount:

| Field | Description |
|-------|-------------|
| `name` | Name of the ConfigMap or Secret |
| `destDir` | Mount the whole ConfigMap or Secret at this directory |
| `destFile` | Mount a single key at this file path. The key mounted is the file's base name |
| `link` | When `true`, symlink the mounted file or directory into the repository working directory (default: `false`) |

```yaml
configMaps:
  - name: app-config            # mounts the whole ConfigMap at /configmaps/app-config
  - name: build-tools
    destDir: /tools             # mounts the whole ConfigMap at /tools instead
secrets:
  - name: api-keys
    destFile: api-keys.json     # mounts only the api-keys.json key at /workspace/api-keys.json
  - name: deploy
    destDir: ci                 # mounts at /workspace/ci
    link: true                  # also symlinks it into the repo working directory at ci/
```

An `env` value can be a literal string or a reference to a key in a Kubernetes Secret. For a literal, set the value directly:

```yaml
env:
  LOG_LEVEL: debug
```

To source a value from a Secret, provide `secretKeyRef` with the Secret name and key. The container receives the Secret's value as the environment variable:

```yaml
env:
  API_KEY:
    secretKeyRef:
      name: my-secret
      key: api-key
```

`resources` sets the CPU and memory requests and limits for the executor container as [Kubernetes quantities](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/):

```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: "1"
    memory: 1Gi
```

### Remote Credentials

Store credentials as Kubernetes Secrets for remote execution. See [Workflows](../../docs/workflows.md) for setup.

```bash
ralph set remote                          # quickstart: reuses your gh login token
ralph set remote --github-key <key.pem>   # GitHub App private key
ralph set remote --github-token <token>   # GitHub personal access token
```

## Custom Instructions

Create `.ralph/instructions.md` to replace the development steps in the AI prompt. The file supplies the prompt's instruction steps only. The surrounding prompt still carries the selected item, the project file path, the git history, and the report contract. If not present, Ralph's [default steps](../ai/development-item-instructions.md) are used.

The default steps are deliberately generic: they send the agent to the repository's own agent instructions for how project items are read, where code belongs, and how tests are written. Custom instructions replace those steps, so state the standards they should follow.

The prompt tells the agent to write its commit message to `report.md` and to end that message with a completion trailer, a bare `<branch>-<hash>` line, when the item is finished. That trailer is the only way an item is ever marked complete, and it is stated outside the instruction steps, so a custom `instructions.md` cannot drop it.
