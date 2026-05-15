# Ralph

I made a Ralph.

Ralph orchestrates AI coding agents to automate development workflows, from branch creation through pull request submission.

## Features

- 🤖 AI-driven development with OpenCode
- 🔄 Iterative workflows until requirements pass
- 🌿 Automated git operations (branch, commit, push, PR)
- 🚀 Service management (start/stop dev services)
- 🔍 Dry-run mode to preview actions
- 🎯 YAML-based project definitions
- 🐙 Remote execution via Argo Workflows on Kubernetes

## Example

Define your project with requirements:

```yaml
name: user-authentication
description: Add user authentication

requirements:
  - category: backend
    description: Authentication API
    items:
      - Users can register with email and password
      - Users can log in with valid credentials
      - JWT tokens are issued on successful authentication
    passing: false

  - category: frontend
    description: Authentication UI
    items:
      - Users can access login and registration forms
      - Login redirects to dashboard on success
      - Invalid credentials show error messages
    passing: false
```

Run it locally:

```bash
ralph user-authentication.yaml --local
```

Ralph will create a branch, implement each requirement using an AI agent, and open a pull request when all requirements pass.

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

### 4. Configure OpenCode

See [OpenCode authentication docs](https://opencode.ai/docs/cli/#auth) for setup instructions.

## More

- [CLI reference](docs/cli.md)
- [Writing projects](docs/skills/ralph-write-project.md)
- [Remote execution workflows](docs/workflows.md)
- [Configuration reference](docs/config.md)
