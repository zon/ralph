# Agent Instructions

## Coding Standard

**IMPORTANT**: Before writing any code, read [docs/zpecs/architecture.md](docs/zpecs/architecture.md) to understand how to write code in this repository. It covers component placement and directs you to the component map in `specs/architecture.yaml` ([Architecture Format](docs/zpecs/architecture-outline.md)).

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/testing.md](docs/testing.md) to understand patterns, conventions, and the webhook service integration testing strategy.

**WARNING**: Be careful when executing ralph with the `--local` flag, as it will apply changes to the local environment.

## Installed Standards

The documents under [docs/zpecs/](docs/zpecs/README.md) are installed, not authored here — they are synced by the `zpecs` CLI from the [specs repository](https://github.com/zon/specs). Changes to those documents belong in the specs repository, not in this one. Skills for authoring specs, orchestrations, architectures, and projects are likewise installed from the specs repository rather than shipped here.

## Versioning

When bumping the version, update **both** files together:
- `internal/version/VERSION`
- `charts/ralph-webhook/Chart.yaml` (`appVersion` and `version`)

Always do a **patch bump** on the chart `version` field alongside any `appVersion` change.
