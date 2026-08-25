# Agent Instructions

## Coding Standard

**IMPORTANT**: Follow [docs/zpecs/code.md](docs/zpecs/code.md) when writing code.

**IMPORTANT**: Read [docs/zpecs/architecture.md](docs/zpecs/architecture.md) when planning the structure of code.

**IMPORTANT**: Read the [Architecture Format](docs/zpecs/architecture-outline.md) when adding or changing a component.

## Prose Standard

**IMPORTANT**: Follow [docs/zpecs/prose.md](docs/zpecs/prose.md) when writing prose.

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/testing.md](docs/testing.md) to understand patterns, conventions, and the webhook service integration testing strategy.

**WARNING**: Be careful when executing ralph with `--mode local` or the default `--mode worktree`, as they apply changes to the local repository.

## Installed Standards

The documents under [docs/zpecs/](docs/zpecs/README.md) are installed, not authored here — they are synced by the `zpecs` CLI from the [specs repository](https://github.com/zon/specs). Changes to those documents belong in the specs repository, not in this one. Skills for authoring specs, orchestrations, architectures, and projects are likewise installed from the specs repository rather than shipped here.

## Versioning

When bumping the version, update **both** files together:
- `internal/version/VERSION`
- `charts/ralph-webhook/Chart.yaml` (`appVersion` and `version`)

Always do a **patch bump** on the chart `version` field alongside any `appVersion` change.
