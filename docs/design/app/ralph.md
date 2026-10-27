# CLI Design

Ralph is a command line tool that runs project files with an AI coding agent. `ralph run` is the default command. The other commands inspect progress, validate input, prepare a machine, or manage remote workflows.

## Commands

* `ralph run <project>` — run a project to completion and open a pull request
* `ralph loop <slug>` — run a configured step list until nothing is left
* `ralph validate <project>` — check that a project file parses and yields items
* `ralph complete [project]` — print the completion hashes on this branch
* `ralph incomplete <project>` — print the project items that are not complete
* `ralph setup` — confirm the machine, and optionally a namespace, is ready
* `ralph command -- <cmd>` — run a command in a remote workflow
* `ralph list` — list Ralph workflows
* `ralph stop <workflow>` — stop a Ralph workflow
* `ralph logs [workflow]` — print workflow pod logs
* `ralph workflow ...` — container entrypoints used by workflows
* `ralph help config` — show the configuration reference
* `ralph help project` — show the project file guide

`ralph --version` prints the version and build date. `ralph run --version` prints the same.

The `workflow` group is not for people. It holds the entrypoints a container runs after its workspace is prepared: `run`, `loop`, `command`, and `token`. See [Workflow Container Design](../workflow.md).

## Shared Options

* `--mode` selects `local`, `worktree`, or `remote` for `run` and `loop`. See [Run Design](../run.md).
* `--model` and `--variant` override the model and its reasoning-effort variant. See [Config Design](ralph-config.md).
* `--agent` overrides the opencode agent for a run's code-writing prompts.
* `--items` overrides the jq query for commands that resolve project items.
* `--context` and `--namespace` (`-n`) target a Kubernetes cluster.
* `--verbose` prints debug output.
* `--no-notify` suppresses desktop notifications in `run` and `loop`.
* `--follow` (`-f`) streams workflow logs. It applies only to `remote` mode.
* `--debug <branch>` runs Ralph from source inside a remote container.
* `--working-dir` (`-C`) changes directory before a run.

A flag with no meaning in the chosen mode is ignored with a warning or rejected with an error. `--follow` is warned about in `local` and `worktree` modes. `--debug` is rejected there.

## Documentation

Ralph embeds its own reference documents. `ralph help config` prints the [config reference](ralph-config.md). `ralph help project` prints the project file guide. Both come from files in the repository, so the help in a running binary matches its source.

## Related

* [Config Design](ralph-config.md)
* [Project Design](../project.md)
* [Completion Design](../completion.md)
* [Run Design](../run.md)
* [Loop Design](../loop.md)
* [Remote Mode Design](../remote.md)
* [Setup Design](../setup.md)
* [Validate Design](../validate.md)
* [Services Design](../services.md)
* [Argo Design](../argo.md)
