# Workflow Container Design

A remote workflow executes `ralph workflow <subcommand>` inside the Ralph container. The `run`, `loop`, `prompt`, and `command` subcommands prepare the same workspace first. The `token` subcommand configures authentication alone.

## Workspace

Setup runs in order:

1. **GitHub authentication** - configure git HTTPS authentication from credentials mounted at `/secrets/github`. Ralph prefers GitHub App credentials (`app-id` and `private-key`) and falls back to a stored `token`. See [Setup Design](setup.md).
2. **OpenCode credentials** - load `auth.json` from the Secret mounted at `/secrets/opencode` into OpenCode's credential store. A mount without that file loads nothing.
3. **Git identity** - configure the bot name and email, defaulting to `ralph-zon[bot]`.
4. **Clone** - clone the repository at the branch the workflow names into `/workspace/repo`.
5. **Checkout** - check out a target branch when the subcommand names one. Create it from the clone branch when it does not exist on the remote.
6. **Links** - link configured ConfigMap and Secret mounts into the repository working directory.

A workflow with no override mounts `opencode-credentials`, the Secret `ralph setup` writes.

## Subcommands

### `ralph workflow run`

Runs one project. It validates the project path, the config, and the item query, then runs the [local mode](local.md) body. Base branch synchronization, iteration, cleanup, and the pull request all happen in the container. The workflow passes the resolved base branch and item query as arguments, so the container does not resolve them again.

### `ralph workflow loop`

Runs one loop. It receives the slug, steps, and iteration cap from the workflow and runs the same loop body as a local loop.

### `ralph workflow prompt`

Runs one OpenCode prompt. It receives the branch, base, prompt, model, variant, agent, and service choice from the workflow and validates the prompt before workspace setup. It then runs the [Prompt Design](prompt.md) flow in the cloned repository.

### `ralph workflow command`

Runs the supplied command tokens in the cloned repository. It backs `ralph command`. A missing command fails before any work runs.

### `ralph workflow token`

Configures git and GitHub authentication and nothing else. It resolves the repository from `--owner` and `--repo`, or from the git remote. It prefers App credentials and falls back to the stored token.

## Debug

`--debug <branch>` checks out that branch of the Ralph repository into `/workspace/ralph` and writes a wrapper that runs Ralph with `go run`. Every later Ralph invocation in the container uses the wrapper instead of the installed binary.
