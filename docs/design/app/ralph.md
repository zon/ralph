# CLI Design

Ralph is a command line tool that runs projects, loops, and direct prompts with an AI coding agent. `ralph run` is the default command. The other commands inspect progress, validate input, prepare a machine, or manage remote workflows.

## Commands

* `ralph run <project>` - run a project to completion and open a pull request
* `ralph loop <slug>` - run a configured step list until nothing is left
* `ralph prompt [text]` - run one OpenCode prompt from text, a file, or standard input on a prompt branch, rename the branch from the work it did, and open a pull request when it commits
* `ralph validate <project>` - check that a project file parses and yields items
* `ralph complete [project]` - print the completion hashes on this branch
* `ralph incomplete <project>` - print the project items that are not complete
* `ralph setup` - confirm the machine, and optionally a namespace, is ready
* `ralph command -- <cmd>` - run a command in a remote workflow
* `ralph list` - list Ralph workflows
* `ralph stop <workflow>` - stop a Ralph workflow
* `ralph logs [workflow]` - print workflow pod logs
* `ralph workflow ...` - container entrypoints used by workflows
* `ralph help config` - show the configuration reference
* `ralph help project` - show the project file guide

`ralph --version` prints the version and build date. `ralph run --version` prints the same.

The `workflow` group is not for people. It holds the entrypoints a container runs after its workspace is prepared: `run`, `loop`, `prompt`, `command`, and `token`. See [Workflow Container Design](../workflow.md).

## Shared Options

* `--mode` selects `local`, `worktree`, or `remote` for `run`, `loop`, and `prompt`. See [Run Design](../run.md).
* `--base` (`-B`) sets the base branch for `run` and `prompt`.
* `--model` and `--variant` override the model and its reasoning-effort variant for `run`, `loop`, and `prompt`. See [Config Design](ralph-config.md).
* `--agent` overrides the OpenCode agent for code-writing prompts in `run`, `loop`, and `prompt`.
* `--items` overrides the jq query for commands that resolve project items.
* `--no-services` skips service startup for `run` and `prompt`.
* `--context` and `--namespace` (`-n`) target a Kubernetes cluster.
* `--verbose` prints debug output.
* `--no-notify` suppresses desktop notifications in `run`, `loop`, and `prompt`.
* `--follow` (`-f`) streams workflow logs. It applies only to `remote` mode. `ralph prompt` and `ralph command` follow by default, so they take `--no-follow` instead.
* `--debug <branch>` runs Ralph from source inside a remote container for `run`, `loop`, and `prompt`.
* `--working-dir` (`-C`) changes directory before a run.

Ralph warns about a flag with no meaning in the chosen mode, or rejects it with an error. In `local` and `worktree` modes, it warns about `--follow` and rejects `--debug`.

## Documentation

Ralph embeds its own reference documents. `ralph help config` prints the [config reference](ralph-config.md). `ralph help project` prints the project file guide. Both come from files in the repository, so the help in a running binary matches its source.
