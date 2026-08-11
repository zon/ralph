# CLI Reference

Ralph is a command-line tool that runs AI-driven development workflows defined in project files — any YAML or JSON file containing an array of work items.

## ralph \<project-file\>

The main command runs a complete development workflow for a project.

```bash
ralph my-feature.yaml
ralph my-feature.yaml --items '.requirements'
```

### Project Steps

1. Creates branch `ralph/<slug>` — from the file's `slug` field, or its base name
2. Resolves the item array with the [item query](formats/project.md#item-query)
3. Iterates until every item is recorded complete
4. Optionally cleans up the project file in its own commit
5. Creates a pull request

### Iteration Steps

For each iteration:

1. Reads the branch commit log to determine which items are already complete
2. Runs `before` commands from `.ralph/config.yaml`
3. Starts services from `.ralph/config.yaml`
4. The picker agent selects one incomplete item
5. The AI implements that item and writes its commit message to `report.md`, ending with `Ralph item <index> (<key>) completed` if the item is finished
6. Commits changes using that message
7. Stops services

See [Iterations](iterations.md) for the completion model.

### Flags

| Flag | Description |
|------|-------------|
| `--items` | jq query selecting the item array (default: config `items`, else `.`) |
| `--cleanup` | Delete the project file in its own commit once complete |
| `--extra-iterations` | Iterations allowed beyond the item count |
| `--once` | Run one iteration without branching or PR |
| `--local` | Run on this machine instead of submitting remotely |
| `--watch` | Submit remotely and monitor progress |
| `-B, --base` | Override base branch for PR creation |
| `--no-services` | Skip service management |

## ralph get

Inspects completion state. Both subcommands emit JSON on stdout, are read-only, and make no AI calls — they are what the picker agent is built from, and they are the way to check a run's progress from a script or by hand.

### ralph get complete

```bash
ralph get complete
ralph get complete projects/csv-export.yaml
```

Reads the commit messages on the current branch that are not on the base branch, parses the [completion trailers](iterations.md#recording-completion), and prints the indices of the completed items as a JSON array — ascending and deduplicated:

```json
[0, 2, 3]
```

The project file is optional. When given, trailers whose index is outside the resolved item array are dropped; without it, every trailer found in the log is reported. Prints `[]` and exits 0 when nothing is complete.

### ralph get incomplete

```bash
ralph get incomplete projects/csv-export.yaml
```

Resolves the item array, removes the items reported by `ralph get complete`, and prints what is left as a JSON array of the items themselves:

```json
[
  {"slug": "export-endpoint", "description": "Expose the export over HTTP"},
  {"slug": "export-error-handling", "description": "Export fails gracefully"}
]
```

Pass `--index` to get the indices instead, in the same form `ralph get complete` uses:

```bash
$ ralph get incomplete projects/csv-export.yaml --index
[1, 4]
```

An empty array means every item is complete — the condition that ends the iteration loop.

### Flags

| Flag | Description |
|------|-------------|
| `--items` | jq query selecting the item array (default: config `items`, else `.`) |
| `-B, --base` | Base branch bounding the commit log (default: config `defaultBranch`) |
| `--index` | `incomplete` only: emit indices rather than items |

## ralph validate

```bash
ralph validate ./projects/my-feature.yaml
ralph validate ./projects/my-feature.yaml --items '.requirements'
```

Checks that the file parses as YAML or JSON, that the item query evaluates against it, and that it resolves to at least one item. On a parse failure it runs a bounded AI fix loop, then rewrites the file as canonical YAML. There is no schema check.

### Flags

| Flag | Description |
|------|-------------|
| `--items` | jq query selecting the item array (default: config `items`, else `.`) |

`--items` resolves the same way it does for a run: the flag first, then `items` in `.ralph/config.yaml`, then `.`. Validate with the query the run will use — a file that validates under `.` and runs under `.requirements` has not been checked.

Note that a successful validation rewrites the file in canonical YAML, and converts a `.json` input to `.yaml`. That is fine for project files you own, but it is not what you want on a file borrowed from another tool; skip validate for those and confirm the query with `ralph get incomplete` instead, which only reads.

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
