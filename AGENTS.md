# Agent Instructions

## Coding Standard

**IMPORTANT**: Before writing any code, read the [coding standard](https://raw.githubusercontent.com/zon/code/refs/heads/main/README.md) to understand the conventions and best practices that must be followed.

## Writing Project Files

**IMPORTANT**: Before writing any project files, read [docs/projects.md](docs/projects.md) to understand the proper format and best practices.

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/testing.md](docs/testing.md) to understand patterns, conventions, and the webhook service integration testing strategy.

**WARNING**: Be careful when executing ralph with the `--local` flag, as it will apply changes to the local environment.

## Versioning

When bumping the version, update **both** files together:
- `internal/version/VERSION`
- `charts/ralph-webhook/Chart.yaml` (`appVersion` and `version`)

Always do a **patch bump** on the chart `version` field alongside any `appVersion` change.
