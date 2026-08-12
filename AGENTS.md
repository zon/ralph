# Agent Instructions

## Reading Projects

**IMPORTANT**: Before implementing a project item, read [docs/specs/project.md](docs/specs/project.md) to understand the conventional item shape — what `items`, `scenarios`, `code`, and `tests` entries mean and what satisfying each one requires. The mechanics ralph itself implements — the item query, item indices, and the completion trailer — are in [docs/projects.md](docs/projects.md).

## Coding Standard

**IMPORTANT**: Before writing any code, read [docs/specs/code.md](docs/specs/code.md) to understand how to write code in this repository. It covers module placement and directs you to the module map in `specs/architecture.yaml` ([Architecture Format](docs/specs/architecture.md)), including a feature's own architecture document when the project names a `feature`.

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/specs/testing.md](docs/specs/testing.md) to understand patterns, conventions, and the webhook service integration testing strategy.

**WARNING**: Be careful when executing ralph with the `--local` flag, as it will apply changes to the local environment.

## Installed Standards

The documents under [docs/specs/](docs/specs/README.md) are installed, not authored here — `just install` copies them from a checkout of the [specs repository](https://github.com/zon/specs), fetching from it when no checkout is available. Changes to those documents belong in the specs repository, not in this one. Skills for authoring specs, orchestrations, architectures, and projects are likewise installed from the specs repository rather than shipped here.

## Versioning

When bumping the version, update **both** files together:
- `internal/version/VERSION`
- `charts/ralph-webhook/Chart.yaml` (`appVersion` and `version`)

Always do a **patch bump** on the chart `version` field alongside any `appVersion` change.
