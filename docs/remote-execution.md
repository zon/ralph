# Remote Execution with Argo Workflows

Ralph supports remote execution of projects using Argo Workflows on Kubernetes. This enables running AI-driven development workflows in a containerized environment with proper isolation and resource management.

## Overview

Remote execution generates and submits an Argo Workflow that:
- Clones your git repository
- Checks out the current branch
- Runs ralph with your project file in a container
- Creates branches and pull requests just like local execution

## Prerequisites

1. **Kubernetes cluster** with Argo Workflows installed
2. **kubectl** configured with access to your cluster
3. **Argo CLI** installed ([argo-workflows.readthedocs.io](https://argo-workflows.readthedocs.io/en/stable/cli/))
4. **Credentials configured** (see Configuration section below)

### Installing Argo Workflows

```bash
# Install Argo Workflows on your cluster
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml
```

### Installing Argo CLI

```bash
# macOS
brew install argo

# Linux
curl -sLO https://github.com/argoproj/argo-workflows/releases/latest/download/argo-linux-amd64.gz
gunzip argo-linux-amd64.gz
chmod +x argo-linux-amd64
sudo mv argo-linux-amd64 /usr/local/bin/argo
```

## Configuration

### 1. Configure Git Credentials

Generate an SSH key pair and configure it for git operations:

```bash
ralph config git
```

This command:
- Generates an Ed25519 SSH key pair
- Creates a Kubernetes Secret with the private key
- Outputs the public key
- Provides a link to add the key to GitHub

Add the public key to your GitHub account at the provided link.

### 2. Configure GitHub Credentials

Set up a GitHub personal access token for creating PRs:

```bash
ralph config github
```

This command:
- Prompts for your GitHub personal access token
- Creates a Kubernetes Secret with the token
- Provides a link to create a new token if needed

The token needs these permissions:
- `repo` (full control of private repositories)
- `workflow` (update GitHub Action workflows)

### 3. Configure OpenCode Credentials

Configure OpenCode AI credentials:

```bash
ralph config opencode
```

This command:
- Reads your OpenCode auth configuration from `~/.local/share/opencode/auth.json`
- Creates a Kubernetes Secret with all configured AI providers
- Provides feedback on success or errors

### 4. Configure Workflow Settings (Optional)

Create or update `.ralph/config.yaml` to customize workflow execution:

```yaml
workflow:
  # Container image configuration
  image:
    repository: ghcr.io/zon/ralph  # Default image location
    tag: latest                    # Image tag to use

  # Kubernetes context and namespace
  context: my-cluster    # Optional: kubectl context to use
  namespace: argo        # Optional: namespace for workflows (default: argo)

  # Additional ConfigMaps to mount
  configMaps:
    - name: my-config
      mountPath: /config

  # Additional Secrets to mount
  secrets:
    - name: my-secret
      mountPath: /secrets

  # Environment variables
  env:
    DEBUG: "true"
    LOG_LEVEL: "info"
```

All workflow settings are optional. Ralph uses sensible defaults if not specified.

## Usage

### Basic Remote Execution

Submit a workflow and return immediately:

```bash
ralph my-feature.yaml
```

This:
- Generates an Argo Workflow YAML
- Submits it to your Kubernetes cluster
- Returns the workflow name
- Disables notifications (since you're not watching)

### Remote Execution with Monitoring

Submit a workflow and watch its progress:

```bash
ralph my-feature.yaml --watch
```

This:
- Generates and submits the workflow
- Monitors execution in real-time
- Shows logs from the workflow pod
- Exits when the workflow completes

### Checking Workflow Status

View running workflows:

```bash
argo list
```

View workflow details:

```bash
argo get <workflow-name>
```

View workflow logs:

```bash
argo logs <workflow-name>
```

Delete a workflow:

```bash
argo delete <workflow-name>
```

## Default Container Image

Ralph provides a default container image at `ghcr.io/zon/ralph:latest` that includes:

- **ralph binary**: The compiled ralph executable
- **Go toolchain**: For building Go applications (version 1.25)
- **Bun runtime**: JavaScript/TypeScript runtime and package manager
- **Playwright**: Browser automation with Chromium, Firefox, and WebKit

### Image Contents

The default image is based on Ubuntu 24.04 and includes:
- System utilities: git, ssh, curl, ca-certificates
- Development tools: Go compiler, Bun runtime
- Browser testing: Playwright with all major browsers and dependencies

### Building the Default Image

To build and push your own version:

```bash
# Set custom repository and tag
export RALPH_IMAGE_REPOSITORY=myregistry.io/myuser/ralph
export RALPH_IMAGE_TAG=v1.0.0

# Build and push
./scripts/push-image.sh
```

The script:
1. Builds the multi-stage Containerfile
2. Tags the image
3. Pushes to the configured registry

### Using a Custom Image

Override the default image in `.ralph/config.yaml`:

```yaml
workflow:
  image:
    repository: myregistry.io/myuser/ralph
    tag: custom-v1
```

Your custom image should include:
- ralph binary (or be able to download it)
- git and ssh for repository operations
- Any runtime dependencies your project needs

## How It Works

When you run ralph in remote mode (the default), here's what happens:

### 1. Workflow Generation

Ralph generates an Argo Workflow YAML that includes:
- **Parameters**: Project file, config, and instructions embedded as workflow parameters
- **Secrets**: Git SSH key, GitHub token, and OpenCode credentials mounted automatically
- **ConfigMaps/Secrets**: Any additional mounts from your config
- **Environment**: Any custom environment variables from your config

### 2. Git Operations

The workflow:
- Clones your repository using the SSH key
- Checks out your current branch
- Runs ralph in the container
- Creates branches and PRs using the GitHub token

### 3. Execution

The workflow:
- Writes embedded parameters to disk (project file, config, instructions)
- Runs `ralph <project-file>` with the embedded configuration
- Follows the normal ralph workflow (iterate, commit, PR)

### 4. Cleanup

After completion:
- **Workflows** are automatically deleted after 1 day (TTL)
- **Pods** are deleted immediately after workflow completion (podGC)
- **Git repository** uses volatile filesystem (emptyDir volume)
- No persistent storage is created

## Workflow Parameters

The generated workflow includes these parameters:

- `project-file`: Your project YAML file content
- `config-yaml`: Your `.ralph/config.yaml` (if it exists)
- `instructions-md`: Your `.ralph/instructions.md` (if it exists)

These are embedded into the workflow and written to the container filesystem at runtime.

## Security

### Credentials Storage

All credentials are stored as Kubernetes Secrets:
- `git-credentials`: SSH private key for git operations
- `github-credentials`: GitHub personal access token
- `opencode-credentials`: OpenCode auth.json with AI provider tokens

### Context and Namespace

Use `--context` and `--namespace` flags to target specific clusters:

```bash
ralph config git --context production --namespace argo
ralph config github --context production --namespace argo
ralph config opencode --context production --namespace argo
```

Or configure in `.ralph/config.yaml`:

```yaml
workflow:
  context: production
  namespace: argo
```

### Secret Mounting

The workflow automatically mounts:
- Git SSH key to `/root/.ssh/id_ed25519`
- GitHub token to `/root/.config/gh/token`
- OpenCode auth.json to `~/.local/share/opencode/auth.json`

## Troubleshooting

### Workflow Submission Fails

```bash
# Check if argo CLI is installed
argo version

# Check kubectl access
kubectl get workflows -n argo

# Check if secrets exist
kubectl get secrets -n argo
```

### Credentials Issues

```bash
# Verify git credentials
kubectl get secret git-credentials -n argo -o yaml

# Verify GitHub credentials  
kubectl get secret github-credentials -n argo -o yaml

# Verify OpenCode credentials
kubectl get secret opencode-credentials -n argo -o yaml
```

### Workflow Execution Fails

```bash
# Check workflow status
argo get <workflow-name>

# View logs
argo logs <workflow-name>

# Describe workflow for events
kubectl describe workflow <workflow-name> -n argo
```

### Image Pull Issues

If the default image fails to pull:

```bash
# Check if image exists
podman pull ghcr.io/zon/ralph:latest

# Use a custom image
# Add to .ralph/config.yaml:
workflow:
  image:
    repository: your-registry/ralph
    tag: your-tag
```

## Examples

### Basic Workflow

```bash
# Configure credentials once
ralph config git
ralph config github
ralph config opencode

# Run project remotely with monitoring
ralph my-feature.yaml --watch
```

### Custom Image and Environment

```yaml
# .ralph/config.yaml
workflow:
  image:
    repository: myregistry.io/ralph-custom
    tag: v2.0.0
  env:
    DEBUG: "true"
    NODE_ENV: "production"
  namespace: my-namespace
```

```bash
ralph my-feature.yaml --watch
```

### Multiple Clusters

```bash
# Development cluster
ralph config git --context dev --namespace argo-dev
ralph my-feature.yaml --watch

# Production cluster  
ralph config git --context prod --namespace argo-prod
ralph my-feature.yaml --watch
```

## Limitations

- Remote execution is incompatible with `--once` flag (use `--local --once` instead)
- The `--watch` flag is not applicable with `--local` flag
- Notifications are disabled in remote mode without `--watch`
- Workflow cleanup happens after 1 day (not configurable)

## Further Reading

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Writing Projects](projects.md)
