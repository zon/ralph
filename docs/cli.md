# CLI Reference

Ralph is a command-line tool that runs AI-driven development workflows defined in project files: any YAML or JSON file containing an array of work items.

## ralph \<project-file\>

The main command runs a complete development workflow for a project.

```bash
ralph my-feature.yaml
ralph my-feature.yaml --items '.requirements'
```

### Project Steps

1. Creates branch `ralph/<slug>` from the file's `slug` field, or its base name
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
5. The AI implements that item and writes its commit message to `report.md`, ending with a bare `<branch>-<hash>` completion trailer if the item is finished
6. Commits changes using that message
7. Stops services

See [Iterations](iterations.md) for the completion model.

### Flags

| Flag | Description |
|------|-------------|
| `-i, --items` | jq query selecting the project item list (default: `.`) |
| `--cleanup` | Delete the project file in its own commit once complete |
| `--extra` | Extra iterations beyond project item count (default: 20% item count) |
| `--once` | Run one iteration without branching or PR |
| `--mode` | Execution mode: `local`, `worktree` (default), or `remote` |
| `--watch` | Submit remotely and monitor progress |
| `-B, --base` | Override base branch for PR creation |
| `--no-services` | Skip service management |
| `--context` | The name of the Kubernetes context to use |
| `-n, --namespace` | The name of the Kubernetes namespace to use |

## ralph loop

`ralph loop` runs an AI iteration over a set of steps. Resolve the steps from a named [loop](../internal/config/config.md#loops) entry in `.ralph/config.yaml` by slug, or pass them directly with the repeated `--step` flag:

```bash
ralph loop update-deps
ralph loop --step "check for outdated dependencies" --step "update the dependency manifest"
```

The execution mode resolves like a run: the `--mode` flag, then the `mode` field in `.ralph/config.yaml`, then `worktree`. Local mode runs the loop in the current checkout. Worktree mode runs it in a sibling git worktree on the `loop-<slug>` branch. Remote mode submits an Argo Workflow so the loop runs inside the workflow container, with `--follow` streaming its logs.

### Flags

| Flag | Description |
|------|-------------|
| `--step` | Step to run in the loop (repeatable) |
| `--max` | Maximum number of iterations (default: 20) |
| `--verbose` | Enable verbose logging |
| `--mode` | Execution mode: `local`, `worktree` (default), or `remote` |
| `-f, --follow` | Follow workflow logs after submission (only with `--mode remote`) |
| `--no-notify` | Disable desktop notifications |
| `-m, --model` | The model to use in format of provider/model |
| `--context` | The name of the Kubernetes context to use |
| `-n, --namespace` | The name of the Kubernetes namespace to use |

## ralph command

`ralph command` runs a single command inside a remote Ralph workflow container and streams its logs. The container clones the current branch, so the command runs against the checked-out branch:

```bash
ralph command go test ./...
```

### Flags

| Flag | Description |
|------|-------------|
| `--no-follow` | Skip following workflow logs |
| `--verbose` | Enable verbose logging |
| `--context` | The name of the Kubernetes context to use |
| `-n, --namespace` | The name of the Kubernetes namespace to use |

## ralph complete

```bash
ralph complete
ralph complete projects/csv-export.yaml
```

Reads the commit messages on the current branch that are not on the base branch, parses the [completion trailers](iterations.md#recording-completion), and prints the completion hashes of the finished items, one per line, sorted and deduplicated. Only trailers naming the current branch count. A trailer naming any other branch is ignored without a warning:

```bash
IYAWN02
9d8LxCD
```

Pass `--json` to receive the same hashes as a JSON array:

```bash
$ ralph complete --json
["IYAWN02","9d8LxCD"]
```

The project file is optional. When given, trailers whose hash matches no resolved item are dropped. Without it, every current-branch trailer found in the log is reported. Prints nothing and exits 0 when nothing is complete; with `--json` an empty list prints as `[]`.

`ralph complete` is read-only and makes no AI calls. Together with `ralph incomplete` it is what the picker agent is built from, and the way to check a run's progress from a script or by hand.

### Flags

| Flag | Description |
|------|-------------|
| `-i, --items` | jq query selecting the project item list (default: `.`) |
| `-B, --base` | Base branch bounding the commit log (default: config `defaultBranch`) |
| `--json` | Print the hashes as a JSON array |

## ralph incomplete

```bash
ralph incomplete projects/csv-export.yaml
```

Resolves the item array, removes the items reported by `ralph complete`, and prints what is left as a JSON array of the items themselves:

```json
[
  {"slug": "export-endpoint", "description": "Expose the export over HTTP"},
  {"slug": "export-error-handling", "description": "Export fails gracefully"}
]
```

An empty array means every item is complete: that condition ends the iteration loop. Like `ralph complete`, it is read-only and makes no AI calls.

### Flags

| Flag | Description |
|------|-------------|
| `-i, --items` | jq query selecting the project item list (default: `.`) |
| `-B, --base` | Base branch bounding the commit log (default: config `defaultBranch`) |

## ralph validate

```bash
ralph validate ./projects/my-feature.yaml
ralph validate ./projects/my-feature.yaml --items '.requirements'
```

Checks that the file parses as YAML or JSON, that the item query evaluates against it, and that it resolves to at least one non-empty item. On a parse failure it runs a bounded AI fix loop, then rewrites the file as canonical YAML. There is no schema check.

### Flags

| Flag | Description |
|------|-------------|
| `-i, --items` | jq query selecting the project item list (default: `.`) |

`-i, --items` resolves the same way it does for a run: the flag first, then `items` in `.ralph/config.yaml`, then `.`. Validate with the query the run will use. A file that validates under `.` and runs under `.requirements` has not been checked.

A successful validation rewrites the file in canonical YAML, and converts a `.json` input to `.yaml`. That is fine for project files you own, but it is not what you want on a file borrowed from another tool. Skip validate for those and confirm the query with `ralph incomplete` instead, which only reads.

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
| `-m, --model` | The model to use in format of provider/model |
| `-B, --base` | Override base branch for PR creation |
| `--mode` | Execution mode: `local`, `worktree` (default), or `remote` |
| `--verbose` | Enable verbose logging |
| `--context` | The name of the Kubernetes context to use |

## ralph help config

```bash
ralph help config
```

Shows the [configuration reference](../internal/config/config.md) for `.ralph/config.yaml`. Interactive terminals render the markdown and page it with the usual pager keys (`↑`/`↓` or `j`/`k` scroll, `space`/`pgdn` and `pgup` page, `q` quits); redirected output prints plain text.

## Other Commands

### ralph setup

```bash
ralph setup --github-key <key.pem>
ralph setup --github-token <token>
ralph setup
```

Configures the Kubernetes credentials needed for remote execution in one shot: a GitHub identity and the OpenCode AI credentials. See [Workflows](workflows.md) for setup.

GitHub credentials accept either a GitHub App private key or a personal access token. A GitHub App is recommended for teams; a personal access token is the quickest way to start. When neither is provided and no existing Secret is found, Ralph falls back to the token from your `gh` login, then to `GITHUB_TOKEN`.

The OpenCode credentials are read from `~/.local/share/opencode/auth.json`.

| Flag | Description |
|------|-------------|
| `--github-key` | Path to a GitHub App private key (`.pem` file) |
| `--github-token` | GitHub personal access token |
| `--context` | The name of the Kubernetes context to use |
| `-n, --namespace` | The name of the Kubernetes namespace to use |

Use `--context` and `--namespace` to target a specific cluster:

```bash
ralph setup --context production --namespace argo
```
