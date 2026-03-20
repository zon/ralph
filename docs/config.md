# Configuration

Ralph looks for `.ralph/config.yaml` in your project root for optional settings.

## Format

```yaml
maxIterations: 10              # Max iterations before stopping (default: 10)
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
  labels:                      # Kubernetes labels for workflow pods (optional)
    environment: production
    team: platform
```

**Note:** API keys are managed by OpenCode, not Ralph. Configure them with `opencode auth`.

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
| `env` | Environment variables to set in the container |
| `labels` | Kubernetes labels to apply to workflow pods |

### Remote Credentials

Store credentials as Kubernetes Secrets for remote execution. See [Workflows](workflows.md) for setup.

```bash
ralph config git        # SSH key for git operations
ralph config github     # GitHub personal access token
ralph config opencode   # OpenCode AI provider tokens
ralph config pulumi     # Pulumi access token
```

## Custom Instructions

Create `.ralph/instructions.md` to guide the AI. Ralph includes this file in the AI prompt automatically. If not present, [default instructions](../internal/config/default-instructions.md) are used.

**Note:** The default instructions include important guidance for requirement management and reporting. Edit carefully to preserve this functionality.
