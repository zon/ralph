# Agent Instructions

## Inline Help

**IMPORTANT**: Follow [docs/inline-help.md](docs/inline-help.md) when writing doc files built into the Ralph app: the markdown documents that Ralph embeds in its binary and renders as help inside the running app, such as `internal/config/config.md` behind `ralph help config`.

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/testing.md](docs/testing.md) to understand patterns and conventions.

**WARNING**: Be careful when executing ralph in the default `--mode local` or in `--mode worktree`, as they apply changes to the local repository.

## Installed Standards

The documents under [docs/zpecs/](docs/zpecs/README.md) are installed, not authored here. They are synced by the `zpecs` CLI from the [specs repository](https://github.com/zon/specs). Changes to those documents belong in the specs repository, not in this one. Skills for authoring specs, orchestrations, architectures, and projects are likewise installed from the specs repository rather than shipped here.

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
