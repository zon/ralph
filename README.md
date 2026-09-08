# Ralph

I made a [Ralph](https://ghuntley.com/ralph/).

Ralph runs coding projects. Ralph can work on a local Git repo or run in a remote [Argo Workflow](https://argoproj.github.io/workflows/). Ralph works like a developer checking out a Git branch, making commits, and submitting a pull request when it's done.

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

## Additional features

- 🚀 Service management: run dev services required by the project
- 🚧 Blocked: Ralph can ask for help by committing a `blocked.md` file when it reaches a dead end
- 🔁 `ralph loop`: run the same steps repeatedly until nothing needs to be done

## Installation

Ralph ships as a Homebrew formula. The formula installs Ralph and the CLI commands it runs — Git, GitHub CLI, OpenCode, kubectl, and the Argo CLI — and tracks the newest tagged release.

OpenCode and the Argo CLI live outside Homebrew core, so add their taps before installing:

```bash
brew tap anomalyco/tap
brew tap argoproj/tap
```

```bash
brew install https://raw.githubusercontent.com/zon/ralph/main/Formula/ralph.rb
```

Install from source instead? Follow the [manual install guide](docs/manual-install.md).

Then authenticate GitHub:

```bash
gh auth login
```

And configure OpenCode. See [OpenCode authentication docs](https://opencode.ai/docs/cli/#auth).

## Help

Ralph contains its own documentation. Read it yourself:

```bash
ralph --help
```

Or add these instructions to your `AGENTS.md` file:

```markdown
We use Ralph to run coding projects. Run `ralph --help` to learn more.
```
