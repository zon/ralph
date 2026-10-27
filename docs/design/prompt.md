# Prompt Design

`ralph prompt` runs one user prompt on a branch named from the prompt slug and streams the OpenCode output. It is the companion of [run](run.md) and [loop](loop.md): no project items, no iteration, and no report contract. The prompt owns its commits. If the pass leaves commits, Ralph renames the branch to the work it actually did, pushes the branch, and opens a pull request.

## Invocation

`ralph prompt <text>` joins one or more positional tokens with single spaces. `ralph prompt --file <path>` reads the prompt from a file. Using positional text and `--file` together fails. With neither, Ralph reads standard input when it is not a terminal. `--file -` reads standard input explicitly, even from a terminal. A missing, empty, or whitespace-only prompt fails before any mode work starts. A prompt that starts with `-` needs `--` before it.

The command accepts `--mode`, `--base` (`-B`), `--model`, `--variant`, `--agent`, `--context`, `--namespace`, `--no-services`, `--no-follow`, `--no-notify`, `--debug`, and `--verbose`. It accepts no project or item options.

## Resolution

Mode follows the same order as [run](run.md): the `--mode` flag, then `mode` in [config](app/ralph-config.md), then `local`.

The base branch follows the same order as a run: `--base`, then the current branch when it differs from the prompt branch, then `defaultBranch` in [config](app/ralph-config.md).

Model, variant, and agent follow [AI Design](ai.md): the flag, then the config field, then OpenCode's default. Unlike a supporting prompt, a direct prompt uses the configured agent because it may write repository code.

The prompt is used as written, so [custom instructions](app/ralph-config.md#custom-instructions) do not apply. The Kubernetes options apply only to remote mode.

## Flow

A prompt runs the before commands, checks out the prompt branch, synchronizes the base branch, starts the services, and runs the pass. When the pass leaves commits, Ralph renames the branch from the commit log, synchronizes the base branch again, pushes the branch, opens the pull request, and, in local mode, restores the starting branch.

## Branch

Ralph asks the AI for a short slug from the prompt and uses it as the branch name, cleaned up like a project branch. A failed slug proposal fails the command before any branch or service work. The branch follows the creation and checkout behavior of a run.

Before the pull request, Ralph asks the AI for a slug from the commit log the pass made and renames the branch to it. The pull request head and title fallback use the new name. A failed proposal or rename fails the command before the push.

* Local mode checks out the prompt branch in the current checkout, creating it from the current branch when it does not exist.
* Worktree mode creates the worktree on the prompt branch with the same origin and local rules as a run, and refuses a branch already checked out in another worktree.
* Remote mode clones the current branch, then checks out the prompt branch in the container, creating it from the clone branch when it does not exist.

## Commit

Ralph does not stage or commit the prompt's working tree and does not choose the prompt's commit messages. The prompt commits according to its own instructions. A prompt that does not ask for a commit may leave changes uncommitted, and that is a valid outcome. Base synchronization can create a merge commit.

Ralph records the branch tip after base synchronization. After a successful pass it compares the tip:

* If the tip did not advance, the prompt made no commits. The command succeeds without a push or pull request. Uncommitted changes remain in a local checkout, and Ralph discards them in worktree and remote modes.
* If the tip advanced, the prompt made commits. Ralph renames the prompt branch from the commit log, synchronizes the base branch again, pushes the branch, generates a pull request description from the commit log with the PR summary prompt, and opens a pull request to the base branch. The summary prompt receives the prompt text in place of a project description. The title is the first H1 in the summary, falling back to the renamed branch.

A failed pass makes no push or pull request. If the summary or pull request step fails, the command fails and the pushed prompt branch remains.

## Environment

Before commands run once before the branch switch, except in worktree mode, where they run inside the new worktree. A non-optional failure fails the command. An optional failure warns and continues.

A service that fails to start or become healthy runs the [service-fix prompt](services.md) before the pass. `--no-services` skips service startup and the fix prompt, and leaves the before commands in place.

The configuration comes from `.ralph/config.yaml` in the checkout. See [Services Design](services.md).

## Pass

The prompt body is one non-interactive OpenCode pass. A prompt slug proposal, a service fix, a base-conflict resolution, or a PR summary can run as supporting passes. The command has no item picker, development prompt, changelog, report contract, block, completion trailer, or iteration recovery. There is no iteration cap. Each invocation is a fresh OpenCode session, not a continued conversation. Cancel a local or worktree pass with Ctrl-C and stop a remote pass with [ralph stop](argo.md).

Ralph reads no report. It does not restrict the tools the prompt can use, so the agent can run git commands itself. The pass's transcript is OpenCode's stdout and stderr. A non-zero OpenCode exit fails the command.

## Remote

Remote mode submits an Argo Workflow and runs the same setup and pass in the [workflow container](workflow.md), following [Remote Mode Design](remote.md). It proposes the slug and resolves the branch and base locally. It requires the current branch to exist on the remote at the same commit, because the container clones that branch. Uncommitted changes are not sent. A prompt from a file or standard input is read locally before submission and is sent as text.

The workflow uses the `ralph-prompt-` name prefix, carries the Ralph label, and uses the configured image, mounts, environment, labels, and resources. It targets the resolved Kubernetes context and namespace, and uses the same TTL and pod garbage collection as the other workflows. It takes the prompt branch lock, as a run does. Ralph embeds the prompt in the workflow manifest as one container argument, so treat it as visible to anyone who can read workflows in the namespace.

Remote mode follows logs by default and waits for the workflow to finish. `--no-follow` submits and returns with the workflow name and an `argo logs` hint. A followed workflow sends a completion notification unless `--no-notify` is set. Stopping the local command does not stop the workflow. A followed failure exits non-zero, and a submission without follow exits zero once the workflow is accepted.

`--debug <branch>` runs Ralph from that source branch inside the container. It is rejected in local and worktree modes. See [Workflow Container Design](workflow.md).

## Container

`ralph workflow prompt` is the container entrypoint. It validates the prompt, prepares the [workspace](workflow.md), and runs the flow above in the cloned checkout. It accepts the branch, base, prompt, model, variant, agent, and `--no-services` from the workflow.
