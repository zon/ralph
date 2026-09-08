# Ralph

I made a [Ralph](https://ghuntley.com/ralph/).

Ralph runs coding projects. A project is any YAML or JSON file with a list of instructions or requirements. Projects run locally, on a [Git Worktree](https://git-scm.com/docs/git-worktree), or isolated in an [Argo Workflow](https://argoproj.github.io/workflows/). Ralph uses [OpenCode](https://opencode.ai/), Git, and the [GitHub CLI](https://cli.github.com/) to branch, code, and submit a pull request when the project is done.

## Projects

A project might look like this:

```yaml
- Reports can be exported as CSV from GET /reports/:id/export
- A request for a missing report ID returns 404
- A malformed report ID returns 400 with an error message
```

Ralph runs a project like this:

1. Check out a branch named after the project
2. Pick the best incomplete project item
3. Run a new OpenCode context with
    - The selected project item
    - Branch Git history
    - A test-based development process (by default)
    - Instructions to report what was done
4. Git commit with the report and push
5. Submit a pull request if all items are complete. Otherwise return to step 2

## Additional features

- 🚀 Service management: run dev services required by the project
- 🔁 `ralph loop`: run the same steps repeatedly until nothing needs to be done

## Installation

### 1. Install Dependencies

- **[Go](https://go.dev/doc/install)**
- **[Git](https://git-scm.com/install/)**
- **[GitHub CLI](https://cli.github.com/)**
- **[OpenCode CLI](https://opencode.ai/docs/#install)**

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

### 5. Optional Remote Workflows

Install if you want to run Argo Workflows:

- **[kubectl](https://kubernetes.io/docs/tasks/tools/)**
- **[Argo Workflows](https://argo-workflows.readthedocs.io/en/latest/installation/)**

Use the setup command to prepare a Kubernetes namespace for Ralph:

```bash
ralph setup --help
```

## Help

Ralph contains its own documentation. Read it yourself:

```bash
ralph --help
```

Or add these instructions to your `AGENTS.md` file:

```markdown
We use Ralph to run coding projects. Run `ralph --help` to learn more.
```
