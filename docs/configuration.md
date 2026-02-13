# Ralph Configuration Guide

This guide explains how to configure ralph for your projects.

## Configuration Files

Ralph uses three types of configuration files:

### 1. Project Files (`.yaml`)

Project files define the development tasks and requirements. The filename (without extension) becomes the git branch name.

**Location**: Anywhere in your project (pass as argument to ralph commands)

**Example**: `projects/add-auth.yaml`

```yaml
name: add-auth
description: Implement user authentication with JWT tokens

requirements:
  - category: backend
    description: User Authentication Model
    steps:
      - Create User model with username, email, and password fields
      - Add bcrypt password hashing in User.setPassword()
      - Add User.verifyPassword() method for authentication
    passing: false
    
  - category: backend
    description: Login and JWT Generation
    steps:
      - Create POST /api/auth/login endpoint
      - Validate username/password credentials
      - Generate JWT token with user ID and expiration
    passing: false
```

**Reference**: See `../slow-choice/projects/*.yaml` for real-world examples.

**Fields**:
- `name` (required): Project name
- `description` (optional): Brief description for PR title
- `requirements` (required): List of requirements to complete
  - `id` (optional): Unique identifier for programmatic access
  - `category` (optional): Requirement category (e.g., backend, frontend, testing)
  - `name` (optional): Legacy field for simple requirement name
  - `description` (optional): Human-readable description of the requirement
  - `steps` (optional): Array of specific implementation steps
  - `passing` (required): Boolean indicating if requirement is met

### 2. Ralph Configuration (`.ralph/config.yaml`)

Project-specific settings for ralph behavior and services.

**Location**: `.ralph/config.yaml` in your project root

**Example**:

```yaml
maxIterations: 10
baseBranch: main
llmProvider: deepseek
llmModel: deepseek-reasoner

services:
  - name: database
    command: docker
    args: [compose, up, -d, db]
    port: 5432
    
  - name: api-server
    command: npm
    args: [run, dev]
    port: 3000
```

**Fields**:
- `maxIterations` (optional, default: 10): Maximum development iterations
- `baseBranch` (optional, default: "main"): Base branch for PRs
- `llmProvider` (optional, default: "deepseek"): LLM provider
- `llmModel` (optional): Provider-specific model name
  - DeepSeek models: `deepseek-reasoner` (R1, default), `deepseek-chat` (V3), `deepseek-coder`
- `services` (optional): List of services to start/stop
  - `name` (required): Service name
  - `command` (required): Command to execute
  - `args` (optional): Command arguments
  - `port` (optional): TCP port for health checking

**Note**: If `.ralph/config.yaml` doesn't exist, ralph uses default values and skips service management.

### 3. Secrets (`.ralph/secrets.yaml` or `~/.ralph/secrets.yaml`)

API keys for LLM providers.

**Locations** (priority order):
1. `.ralph/secrets.yaml` (project-specific, highest priority)
2. `~/.ralph/secrets.yaml` (global, fallback)

**Example**:

```yaml
apiKeys:
  deepseek: sk-xxxxxxxxxxxxx
  # Add other providers as needed for future multi-provider support
  # anthropic: sk-ant-xxxxxxxxxxxxx
  # openai: sk-xxxxxxxxxxxxx
```

**IMPORTANT**: 
- Add `.ralph/secrets.yaml` to `.gitignore`
- Never commit secrets to version control
- Use project-specific secrets to override global settings

## Dry-Run Mode

All ralph commands support `--dry-run` mode, which simulates execution without making changes:

```bash
# Preview what would happen
ralph run project.yaml --dry-run

# See single iteration plan
ralph once project.yaml --dry-run
```

In dry-run mode, ralph will:
- ✅ Load and validate configurations
- ✅ Display the execution plan
- ✅ Show which services would start
- ✅ Show what git operations would occur
- ❌ NOT start services
- ❌ NOT make git commits/branches
- ❌ NOT call LLM APIs
- ❌ NOT create pull requests

## Configuration Priority

Settings can come from multiple sources with this priority (highest to lowest):

1. **Command-line flags** (e.g., `--max-iterations 5`)
2. **Project config** (`.ralph/config.yaml` in project root)
3. **Default values** (built into ralph)

For secrets:
1. **Project secrets** (`.ralph/secrets.yaml` in project root)
2. **Global secrets** (`~/.ralph/secrets.yaml`)

## Setting Up a New Project

1. **Create project directory structure**:
   ```bash
   mkdir -p .ralph
   ```

2. **Copy example configuration**:
   ```bash
   cp .ralph/config.example.yaml .ralph/config.yaml
   ```

3. **Edit configuration** to match your project needs

4. **Create secrets file** (if not using global):
   ```bash
   cp .ralph/secrets.example.yaml .ralph/secrets.yaml
   # Edit and add your API keys
   ```

5. **Add to .gitignore**:
   ```bash
   echo ".ralph/secrets.yaml" >> .gitignore
   ```

6. **Create a project file**:
   ```bash
   mkdir -p projects
   cat > projects/my-feature.yaml <<EOF
   name: my-feature
   description: Add new feature X
   requirements:
     - passing: false
   EOF
   ```

7. **Run ralph**:
   ```bash
   # Test with dry-run first
   ralph run projects/my-feature.yaml --dry-run
   
   # Run for real
   ralph run projects/my-feature.yaml
   ```

## Examples

### Minimal Project (No Configuration)

```bash
# Create simple project file
cat > task.yaml <<EOF
name: fix-bug
requirements:
  - passing: false
EOF

# Run with defaults (no services, default iterations)
ralph run task.yaml
```

### Full Configuration Example

**.ralph/config.yaml**:
```yaml
maxIterations: 15
baseBranch: develop
llmProvider: deepseek
llmModel: deepseek-reasoner

services:
  - name: postgres
    command: docker
    args: [compose, up, -d, postgres]
    port: 5432
    
  - name: redis
    command: docker
    args: [compose, up, -d, redis]
    port: 6379
    
  - name: dev-server
    command: npm
    args: [run, dev]
    port: 3000
```

**~/.ralph/secrets.yaml**:
```yaml
apiKeys:
  deepseek: sk-your-key-here
```

**projects/feature.yaml**:
```yaml
name: user-profiles
description: Implement user profile pages with avatars

requirements:
  - id: model
    name: Create UserProfile model
    passing: false
    
  - id: api
    name: Add profile API endpoints
    passing: false
    
  - id: ui
    name: Build profile page UI
    passing: false
    
  - id: tests
    name: Write integration tests
    passing: false
```

**Usage**:
```bash
# Single iteration (services start/stop automatically)
ralph once projects/feature.yaml

# Full workflow (branch, iterate until done, create PR)
ralph run projects/feature.yaml

# Skip services for this run
ralph once projects/feature.yaml --no-services

# Override max iterations
ralph run projects/feature.yaml --max-iterations 20
```

## Troubleshooting

### "Failed to load config file"
- Check that `.ralph/config.yaml` exists and has valid YAML syntax
- Use `--verbose` flag for detailed error messages
- Try `--dry-run` to validate configuration without executing

### "No API key found for provider"
- Ensure secrets file exists at `~/.ralph/secrets.yaml` or `.ralph/secrets.yaml`
- Verify API key is set for your configured provider
- Check file permissions (should be readable by you)

### "Service failed to start"
- Verify service command is correct in `.ralph/config.yaml`
- Check that required dependencies are installed
- Look at service logs if available
- Try starting the service manually to debug

### "Port already in use"
- Another instance of the service may be running
- Check with `lsof -i :PORT` or `netstat -an | grep PORT`
- Stop conflicting service or change port in config
