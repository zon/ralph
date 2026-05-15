# Agent Instructions

## Coding Standard

**IMPORTANT**: Before writing any code, read the [coding standard](https://raw.githubusercontent.com/zon/code/refs/heads/main/README.md) to understand the conventions and best practices that must be followed.

## Testing and Execution

**IMPORTANT**: Before writing any tests, read [docs/testing.md](docs/testing.md) to understand patterns, conventions, and the webhook service integration testing strategy.

**WARNING**: Be careful when executing ralph with the `--local` flag, as it will apply changes to the local environment.

## Ralph Skills

When writing or editing a ralph skill in `.claude/skills/`, all references to files in the ralph repository must use markdown links — never backtick code spans or bare paths. This ensures `rewriteLinks` can expand them to absolute raw GitHub URLs when skills are installed into other repositories.

```markdown
<!-- correct -->
Read [docs/formats/specs.md](docs/formats/specs.md) before drafting.

<!-- wrong -->
Read `docs/formats/specs.md` before drafting.
```

References to files in the **target** project (e.g. `./specs/features/...`) do not need links — those paths are intentionally resolved in the project where the skill runs.

## Versioning

When bumping the version, update **both** files together:
- `internal/version/VERSION`
- `charts/ralph-webhook/Chart.yaml` (`appVersion` and `version`)

Always do a **patch bump** on the chart `version` field alongside any `appVersion` change.
