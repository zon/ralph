# Configuration

Ralph looks for `.ralph/config.yaml` in your project root for optional settings.

## Format

```yaml
items: .requirements           # jq query selecting the item array from a project file (default: .)
cleanup: false                 # Delete the project file in its own commit once complete (default: false)
extraIterations:               # Extra iterations beyond item count (unset = 20% of items, rounded up)
defaultBranch: main             # Default branch for PRs (default: main)
model: deepseek/deepseek-chat  # AI model (default: deepseek/deepseek-chat)

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
    tag: latest
  context: my-cluster          # kubectl context (optional)
  namespace: argo              # workflow namespace (default: argo)
  configMaps:                  # additional ConfigMaps to mount (optional)
    - name: my-config
      mountPath: /config
  secrets:                     # additional Secrets to mount (optional)
    - name: my-secret
      mountPath: /secrets
  env:                         # environment variables (optional)
    DEBUG: "true"
    API_KEY:                   # value from a Kubernetes secret
      secretKeyRef:
        name: my-secret
        key: api-key
  labels:                      # Kubernetes labels for workflow pods (optional)
    environment: production
    team: platform
```

**Note:** API keys are managed by OpenCode, not Ralph. Configure them with `opencode auth`.

## Items

`items` is a [jq](https://jqlang.org/manual/) query that selects the array of work items from a project file. It defaults to `.`, which is correct for a project file whose top level is already an array.

```yaml
items: .requirements                                  # nested list
items: .spec.tasks                                    # deeper nesting
items: '.issues | map(select(.state == "open"))'      # filtered
```

The query must resolve to at least one non-empty item; empty outputs — null, `false`, `0`, blank strings, `{}`, `[]` — are dropped before indexing. Every command that reads a project file — the run command, `ralph get`, and `ralph validate` — resolves it the same way: `--items` first, then this field, then `.`. Keep the query stable for the duration of a run; it defines the indices that completion tracking uses. See [Project Format](formats/project.md#item-query) and [Iterations](iterations.md).

## Iterations

`extraIterations` sets how many iterations the loop may run beyond the item count. The limit is `len(items) + extraIterations`. When unset it defaults to 20% of the item count, rounded up. `--extra-iterations` overrides it.

`cleanup` deletes the project file once every item is complete, in a commit of its own, before the pull request is opened. Off by default. `--cleanup` enables it for a single run. Completion history lives in the branch's commit trailers, so cleaning up the file does not lose it.

## Review

`review` configures the `ralph review` command. It defines the standards or guidelines for the AI to review the codebase against.

```yaml
review:
  model: google/gemini-2.5-pro  # optional: AI model override (falls back to root 'model')
  items:
    - text: "All functions must have unit tests."
    - file: docs/coding-standards.md
    - url: https://example.com/style-guide
```

Each item requires exactly one source:

| Field | Description |
|-------|-------------|
| `text` | Inline string content |
| `file` | Path to a file relative to the repo root |
| `url` | HTTP URL returning plain text |

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

- Commands run sequentially and must exit successfully before ralph proceeds (unless marked optional)
- Each entry requires `name` and `command`; `args`, `workDir`, and `optional` are optional
- Set `optional: true` to allow a command to fail without aborting the run (a warning is logged instead)
- Useful for compilation, code generation, dependency installation, database migrations

## Services

`services` defines processes to start before the iteration loop and stop after execution.

- Services are started in order
- Health checks wait for TCP ports to respond if `port` is specified
- Services are stopped gracefully (SIGTERM) after execution
- Use `--no-services` to skip service management

## Workflow

`workflow` configures remote execution on Kubernetes via Argo Workflows. All fields are optional.

| Field | Description |
|-------|-------------|
| `image.repository` | Container image (default: `ghcr.io/zon/ralph`) |
| `image.tag` | Image tag (default: `latest`) |
| `context` | kubectl context to use |
| `namespace` | Kubernetes namespace (default: `argo`) |
| `configMaps` | Additional ConfigMaps to mount |
| `secrets` | Additional Secrets to mount |
| `env` | Environment variables to set in the container; each value is a literal string or a Kubernetes secret reference |
| `labels` | Kubernetes labels to apply to workflow pods |

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

### Remote Credentials

Store credentials as Kubernetes Secrets for remote execution. See [Workflows](workflows.md) for setup.

```bash
ralph config git        # SSH key for git operations
ralph config github     # GitHub personal access token
ralph config opencode   # OpenCode AI provider tokens
ralph config pulumi     # Pulumi access token
```

## Custom Instructions

Create `.ralph/instructions.md` to replace the development steps in the AI prompt. The file supplies the prompt's instruction steps only — the surrounding prompt still carries the selected item, the project file path, the git history, and the report contract. If not present, ralph's [default steps](../internal/ai/development-item-instructions.md) are used.

The default steps are deliberately generic: they send the agent to the repository's own agent instructions for how project items are read, where code belongs, and how tests are written. Custom instructions replace those steps, so state the standards they should follow.

**Note:** The prompt tells the agent to write its commit message to `report.md` and to end that message with a completion trailer, a bare `<branch>-<index>` line, when the item is finished. That trailer is the only way an item is ever marked complete, and it is stated outside the instruction steps, so a custom `instructions.md` cannot drop it.
