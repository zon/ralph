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
