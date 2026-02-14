# Ralph - AI-Powered Development Orchestration

Ralph orchestrates AI coding agents to automate development workflows from branch creation through pull request submission.

## Features

- 🤖 AI-driven development with DeepSeek LLM
- 🔄 Iterative workflows until requirements pass
- 🌿 Automated git operations (branch, commit, push, PR)
- 🚀 Service management (start/stop dev services)
- 🔍 Dry-run mode to preview actions
- 🎯 YAML-based project definitions

## Installation

```bash
go install github.com/zon/ralph/cmd/ralph@latest
```

Ensure `$GOPATH/bin` is in your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## Quick Start

### 1. Install Dependencies

- **OpenCode CLI**: [opencode.ai](https://opencode.ai/docs/cli/)
- **GitHub CLI**: [cli.github.com](https://cli.github.com/)

### 2. Configure OpenCode

See [OpenCode configuration docs](https://opencode.ai/docs/cli/) for setup instructions.

Example:
```bash
opencode config set model deepseek/deepseek-chat
export DEEPSEEK_API_KEY=sk-your-key
```

Get API keys: [DeepSeek](https://platform.deepseek.com/) | [OpenAI](https://platform.openai.com/) | [Anthropic](https://console.anthropic.com/)

### 3. Authenticate GitHub

See [GitHub CLI authentication](https://cli.github.com/manual/gh_auth_login)

### 4. Create a Project File

```bash
cat > projects/my-feature.yaml <<EOF
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

### 5. Run Ralph

```bash
# Preview first
ralph projects/my-feature.yaml --dry-run

# Execute full workflow: branch → iterate → PR
ralph projects/my-feature.yaml
```

## Usage

Run `ralph --help` to see all available options.

### Full Orchestration

```bash
ralph <project-file>
```

Creates branch, iterates development cycles, commits changes, generates PR summary, and creates GitHub pull request.

### Single Iteration Mode

```bash
ralph <project-file> --once
```

Runs a single development iteration without creating branches, committing, or submitting PRs. Stages the project file only and produces a `report.md` file with development results. Useful for local development and testing.

## Configuration

### Project Files

Define requirements in YAML. Filename becomes branch name.

```yaml
name: add-feature
description: Implement feature X

requirements:
  - category: backend
    description: API Endpoint
    steps:
      - Create GET /api/feature endpoint
      - Add validation
      - Return JSON response
    passing: false
```

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
```

**Note:** LLM configuration (model, API keys) is managed by OpenCode, not Ralph.

## Examples

### Basic Workflow

```bash
# Create project file
cat > projects/add-logging.yaml <<EOF
name: add-logging
description: Add structured logging
requirements:
  - passing: false
EOF

# Preview
ralph projects/add-logging.yaml --dry-run

# Execute
ralph projects/add-logging.yaml
```

### With Services

```bash
# Configure services
cat > .ralph/config.yaml <<EOF
services:
  - name: postgres
    command: docker
    args: [compose, up, -d, postgres]
    port: 5432
EOF

# Run - services start/stop automatically
ralph --once projects/my-feature.yaml
```

### Custom Development Instructions

Create `.ralph/instructions.md` to guide the AI:

```markdown
# Development Instructions

## Code Style
- Use functional components in React
- Follow airbnb eslint rules

## Testing
- Write tests for all new endpoints
- Minimum 80% coverage
```

Ralph includes this file in the AI prompt automatically.

## Troubleshooting

**"OpenCode not configured"**

See [OpenCode configuration docs](https://opencode.ai/docs/cli/)

**"Service failed to start"**
- Verify command in `.ralph/config.yaml`
- Check dependencies installed
- Use `--verbose` for details
- Skip with `--no-services`

**"Port already in use"**
```bash
lsof -i :3000  # Find process
kill <PID>     # Stop it
```

**"Branch already exists"**
```bash
# Use different filename or delete existing branch
git branch -D feature-branch
git push origin --delete feature-branch
```

**"gh: not authenticated"**
```bash
gh auth login
```

## How It Works

**`ralph <project-file>`** (full orchestration):
1. Create git branch from filename
2. Iterate: start services → run AI agent → commit changes
3. Continue until requirements pass or max iterations
4. Generate PR summary with AI
5. Push branch and create GitHub PR

**`ralph --once <project-file>`** (single iteration):
1. Start configured services
2. Generate prompt from git history + project requirements + custom instructions
3. Run AI agent to implement changes
4. Stage project file only (other changes remain unstaged)
5. Generate `report.md` with development results
6. Stop services

Note: `--once` mode does not create branches, commit changes, or submit PRs.

## Dependencies

**Required:**
- Go 1.23+
- Git
- GitHub CLI (`gh`)
- DeepSeek API key

**Optional:**
- Docker (for containerized services)
- Node.js/npm (for JS/TS projects)

## Development

```bash
# Build
go build -o bin/ralph ./cmd/ralph

# Test
go test ./... -v

# Test with coverage
go test ./... -cover
```


