# Ralph

I made a Ralph.

Ralph orchestrates AI coding agents to automate development workflows from branch creation through pull request submission.

See [docs/projects.md](docs/projects.md) for writing project files.

## Features

- 🤖 AI-driven development with OpenCode CLI
- 🔄 Iterative workflows until requirements pass
- 🌿 Automated git operations (branch, commit, push, PR)
- 🚀 Service management (start/stop dev services)
- 🔍 Dry-run mode to preview actions
- 🎯 YAML-based project definitions
- ☁️ Remote execution via Argo Workflows on Kubernetes

## Installation

### 1. Install Dependencies

- **Go**: [go.dev/dl](https://go.dev/dl)
- **Git**: [git-scm.com](https://git-scm.com/downloads)
- **GitHub CLI**: [cli.github.com](https://cli.github.com/)
- **OpenCode**: [opencode.ai](https://opencode.ai/docs/cli/)

### 2. Install Ralph

```bash
go install github.com/zon/ralph/cmd/ralph@latest
```

Ensure `$GOPATH/bin` is in your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### 3. Authenticate GitHub

```bash
gh auth login
```

See [GitHub CLI authentication](https://cli.github.com/manual/gh_auth_login)

### 4. Configure OpenCode

See [OpenCode authentication docs](https://opencode.ai/docs/cli/#auth) for setup instructions.

## Usage

```bash
# Preview workflow
ralph my-feature.yaml --dry-run

# Full workflow: branch → iterate → commit → PR
ralph my-feature.yaml

# Single iteration (no branch/commit/PR)
ralph my-feature.yaml --once

# Remote execution on Kubernetes
ralph my-feature.yaml --remote --watch
```

See [docs/projects.md](docs/projects.md) for how to write project files.

See [docs/remote-execution.md](docs/remote-execution.md) for remote execution with Argo Workflows.

## Configuration

### Ralph Config (`.ralph/config.yaml`)

Optional project-specific settings:

```yaml
maxIterations: 10  # Max iterations before stopping
baseBranch: main   # Base branch for PRs

# Optional: Services to manage
services:
  - name: database
    command: podman
    args: [compose, up, -d, db]
    port: 5432  # For health checking
    
  - name: api-server
    command: npm
    args: [run, dev]
    port: 3000

# Optional: Workflow settings for remote execution
workflow:
  image:
    repository: ghcr.io/zon/ralph
    tag: latest
  context: my-cluster
  namespace: argo
```

**Note:** LLM configuration (model, API keys) is managed by OpenCode, not Ralph.

See [docs/remote-execution.md](docs/remote-execution.md) for full workflow configuration options.

### Custom Development Instructions (`.ralph/instructions.md`)

Create `.ralph/instructions.md` to guide the AI. Ralph includes this file in the AI prompt automatically. If not present, [default instructions](internal/config/default-instructions.md) are used.

**Note:** The default instructions include important instructions for requirement management and reporting. Edit carefully to preserve this functionality.
