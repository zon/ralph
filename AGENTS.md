# Agent Instructions

## Inline Help

**IMPORTANT**: Follow [docs/inline-help.md](docs/inline-help.md) when writing doc files built into the Ralph app: the markdown documents that Ralph embeds in its binary and renders as help inside the running app, such as `internal/config/config.md` behind `ralph help config`.

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/testing.md](docs/testing.md) to understand patterns and conventions.

**WARNING**: Be careful when executing ralph in the default `--mode local` or in `--mode worktree`, as they apply changes to the local repository.

## Installed Standards

The documents under [docs/zpecs/](docs/zpecs/README.md) are installed, not authored here. They are synced by the `zpecs` CLI from the [specs repository](https://github.com/zon/specs). Changes to those documents belong in the specs repository, not in this one. Skills for authoring specs, orchestrations, architectures, and projects are likewise installed from the specs repository rather than shipped here.

## Prose

Use hyphens as list separators, like `label - description`. Do not use em dashes.

## Versioning

When bumping the version, update `internal/version/VERSION`.

## Zpecs

* [Milestones](docs/milestones.md) - High level objectives
* [Regressions](regressions.md) - How to organize and resolve bugs and problems found during other work
* [Code](docs/zpecs/code.md) - Read before writing code
  * [Architecture](docs/zpecs/architecture.md) - Read before planning code structure
  * [Dependencies](docs/zpecs/dependencies.md) - Read before adding or changing code dependencies
  * [Containers](docs/zpecs/containers.md) - Read before building or delivering a container image
* [Prose](docs/zpecs/prose.md) - Read before writing any kind of prose
* [Prompts](docs/zpecs/prompts.md) - Read before writing an agent prompt
* [Spec](docs/zpecs/specs.md) - Read before writing a spec
* [Project](docs/zpecs/project.md) - Read before writing a project
* [Glossary](docs/zpecs/glossary.md) - Read when a term is unclear
* [Process](docs/zpecs/process.md) - Read before planning milestones, projects, or regressions
* [Design](docs/zpecs/design.md) - Read before writing a high level design doc

## Design

High level design docs.

* [CLI](docs/design/app/ralph.md) - The command line interface and its shared options
* [Config](docs/design/app/ralph-config.md) - The `.ralph/config.yaml` file and custom instructions
* [Project](docs/design/project.md) - Project files, items, the item query, and metadata
* [Completion](docs/design/completion.md) - Item hashes, completion trailers, and the completion commands
* [Run](docs/design/run.md) - The primary project loop, its modes, commits, and pull request
* [Local Mode](docs/design/local.md) - Running in the current checkout
* [Worktree Mode](docs/design/worktree.md) - Running beside the checkout in a Git worktree
* [Remote Mode](docs/design/remote.md) - Submitting an Argo Workflow
* [Workflow Container](docs/design/workflow.md) - The container workspace and workflow subcommands
* [AI](docs/design/ai.md) - Prompts, model and agent selection, and agent reports
* [Loop](docs/design/loop.md) - The step loop behind `ralph loop`
* [Prompt](docs/design/prompt.md) - One OpenCode pass on a prompt branch with services and an optional pull request
* [Base Branch Sync](docs/design/base-branch-sync.md) - How runs and loops merge their base branch
* [Iteration Recovery](docs/design/iteration-recovery.md) - How a failed iteration is carried forward
* [Kubernetes](docs/design/kube.md) - Context and namespace targeting
* [Setup](docs/design/setup.md) - Local readiness and namespace preparation
* [Validate](docs/design/validate.md) - Project validation and the repair loop
* [Services](docs/design/services.md) - Before commands and per-iteration services
* [Argo](docs/design/argo.md) - Listing, stopping, and reading workflow logs
* [Deployment](docs/design/deployment.md) - The binary, the container image, and credentials
