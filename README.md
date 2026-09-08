# Ralph

I made a [Ralph](https://ghuntley.com/ralph/).

Ralph is a coding agent that runs project files. Ralph can run in a local Git repo or in an [Argo Workflow](https://argoproj.github.io/workflows/). Ralph works like a developer checking out a Git branch, making commits, and submitting a pull request when it's done.

## Projects

A project is any YAML or JSON file with a list of instructions or requirements. A project file might have these contents:

```yaml
- Reports can be exported as CSV from GET /reports/:id/export
- A request for a missing report ID returns 404
- A malformed report ID returns 400 with an error message
```

This command might run the project:

```bash
ralph projects/csv-export.yaml
```

Ralph runs projects like this:

1. Check out a Git branch
2. Pick the best incomplete project item
3. Run a new OpenCode context with
    - The selected project item
    - Branch Git history
    - A test-based development process (by default)
    - Instructions to report what was done
4. Git commit with the report and push
5. Submit a pull request if all items are complete. Otherwise return to step 2

Ralph marks an item complete when the report it commits ends with a bare `<branch>-<hash>` line.

## Additional features

- 🚀 Service management: run dev services required by the project
- 🚧 Blocked: Ralph can ask for help by committing a `blocked.md` file when it reaches a dead end
- 🔁 `ralph loop`: run the same steps repeatedly until nothing needs to be done

## Install

### Homebrew

Taps for Ralph, OpenCode, and Argo is required:

```bash
brew tap zon/tap # Ralph
brew tap anomalyco/tap # OpenCode
brew tap argoproj/tap # Argo
```

Install Ralph and it's dependencies. Includes Git, GitHub CLI, OpenCode, kubectl, and the Argo CLI:

```bash
brew install ralph
```

### Manual

Follow the [manual install instructions](docs/manual-install.md) to install from source.

## Setup

`ralph setup` confirms git, `gh`, and OpenCode are ready to run Ralph. Get them ready first:

```bash
gh auth login
```

And configure OpenCode. See [OpenCode authentication docs](https://opencode.ai/docs/cli/#auth).

## Remote workflows

When you target a Kubernetes namespace, `ralph setup` confirms kubectl and argo are installed, then prepares it for Ralph workflows. Target one with `--namespace` or `workflow.namespace` in `.ralph/config.yaml`. With no targeted namespace, `ralph setup` only confirms the local tools.

## Help

Ralph contains its own documentation. Read it yourself:

```bash
ralph --help
```

Or add these instructions to your `AGENTS.md` file:

> We use Ralph to run coding projects. Run `ralph --help` to learn more.
