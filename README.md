# Ralph

I made a Ralph.

Ralph orchestrates AI coding agents to automate development workflows from branch creation through pull request submission.

## Features

- 🤖 AI-driven development with OpenCode CLI
- 🔄 Iterative workflows until requirements pass
- 🌿 Automated git operations (branch, commit, push, PR)
- 🚀 Service management (start/stop dev services)
- 🔍 Dry-run mode to preview actions
- 🎯 YAML-based project definitions

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

See [OpenCode authentication docs](https://opencode.ai/docs/cli/authentication) for setup instructions.

## Usage

### Create a Project File

Define your development requirements in a YAML file:

```bash
cat > my-feature.yaml <<EOF
name: my-feature
description: Add user authentication

requirements:
  - category: backend
    description: User Authentication Model
    steps:
      - Create User model with username and email fields
      - Add password hashing with bcrypt
      - Add login validation method
    passing: false
EOF
```

### Full Orchestration

Creates branch, iterates development cycles, commits changes, generates PR summary, and creates GitHub pull request.

```bash
# Preview first
ralph my-feature.yaml --dry-run

# Execute full workflow: branch → iterate → PR
ralph my-feature.yaml
```

### Single Iteration Mode

Runs a single development iteration without creating branches, committing, or submitting PRs. Useful for local development and testing changes before committing.

```bash
ralph my-feature.yaml --once
```

## Configuration

### Ralph Config (`.ralph/config.yaml`)

Optional project-specific settings:

```yaml
maxIterations: 10  # Max iterations before stopping
baseBranch: main   # Base branch for PRs

# Optional: Services to manage
services:
  - name: database
    command: docker
    args: [compose, up, -d, db]
    port: 5432  # For health checking
    
  - name: api-server
    command: npm
    args: [run, dev]
    port: 3000
```

**Note:** LLM configuration (model, API keys) is managed by OpenCode, not Ralph.

### Custom Development Instructions (`.ralph/instructions.md`)

Create `.ralph/instructions.md` to guide the AI. Ralph includes this file in the AI prompt automatically. If not present, [default instructions](internal/config/default-instructions.md) are used.
