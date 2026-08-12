# Agent Instructions

## Coding Standard

**IMPORTANT**: Before writing any code, read [docs/specs/code.md](docs/specs/code.md) to understand how to write code in this repository.

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
