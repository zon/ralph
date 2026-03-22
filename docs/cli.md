# CLI Reference

Ralph is a command-line tool that runs AI-driven development workflows defined in project YAML files.

## ralph \<project-file\>

The main command runs a complete development workflow for a project.

```bash
ralph my-feature.yaml
```

### Project Steps

1. Creates branch `ralph/<project-name>`
2. Iterates over requirements
3. Creates a pull request

### Requirement Steps

For each requirement:

1. Runs `before` commands from `.ralph/config.yaml`
2. Starts services from `.ralph/config.yaml`
3. AI implements changes based on the requirement description and items
4. Validates the implementation against each item
5. Commits changes
6. Updates `report.md` with pass/fail status
7. Stops services

### Flags

| Flag | Description |
|------|-------------|
| `--once` | Run one iteration without branching or PR |
| `--local` | Run on this machine instead of submitting remotely |
| `--watch` | Submit remotely and monitor progress |
| `--no-services` | Skip service management |

## ralph review

The `review` command runs an AI-driven code review against standards defined in `.ralph/config.yaml`.

```bash
ralph review
```

### Review Steps

1. Creates branch `ralph/review-YYYY-MM-DD`
2. Iterates over each review item from config
3. AI reviews the codebase against the item's content
4. Commits any changes to the project file
5. Creates a pull request with an AI-generated summary

### Flags

| Flag | Description |
|------|-------------|
| `-p, --project` | Path to output project YAML file (default: `projects/review-YYYY-MM-DD.yaml`) |
| `-m, --model` | Override AI model from config |
| `-B, --base` | Override base branch for PR creation |
| `--local` | Run on this machine instead of submitting remotely |
| `--verbose` | Enable verbose logging |
| `--context` | Kubernetes context to use |

## Other Commands

### ralph config git

```bash
ralph config git
```

Generates an Ed25519 SSH key pair, creates a Kubernetes Secret with the private key, and prints the public key to add to GitHub. Required for remote execution.

### ralph config github

```bash
ralph config github
```

Prompts for a GitHub personal access token and stores it as a Kubernetes Secret. The token needs `repo` and `workflow` permissions.

### ralph config opencode

```bash
ralph config opencode
```

Reads `~/.local/share/opencode/auth.json` and stores it as a Kubernetes Secret with all configured AI providers.

Use `--context` and `--namespace` to target a specific cluster:

```bash
ralph config git --context production --namespace argo
```

### ralph config pulumi

```bash
ralph config pulumi
```

Prompts for a Pulumi access token and stores it as a Kubernetes Secret. The token is required for remote execution with Pulumi-based workflows.

You can provide the token as an argument, via the `PULUMI_ACCESS_TOKEN` environment variable, or enter it interactively when prompted:

```bash
ralph config pulumi <your-token>
ralph config pulumi --context production --namespace argo
```
