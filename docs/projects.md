# Writing Projects

Projects are YAML files that define work for AI agents.

## Format

```yaml
name: project-identifier          # Used for branch naming (ralph/<name>)
description: Brief description    # Used in PR title

requirements:
  - category: backend             # Group related requirements
    description: What to accomplish
    items:
      - Specific outcome 1
      - Specific outcome 2
    passing: false                # false = needs work, true = complete
```

A project can have multiple requirements across different categories. Ralph reads all requirements where `passing: false` and picks the highest priority to work on.

## Naming Projects

The `name` field becomes the branch name: `ralph/<name>`. Use lowercase, hyphen-separated identifiers:

- `user-authentication`
- `fix-pagination`
- `csv-export`

Name your project file to match: `user-authentication.yaml`.

Project files are stored in the `./projects` directory of the repo.

## Writing Requirements

See [Writing Good Requirements](writing-requirements.md) for guidance on writing effective requirements.

## Validating Projects

After writing a project file, validate it with:

```sh
ralph validate <project-file>
```

Fix any reported errors before proceeding.

## Version Bumps

If the repo uses versioning, every project must include a `version` requirement. Specify the bump level — not the target version number. Ralph determines the current version and applies the bump itself.

Each versioned resource is bumped independently based on how its own interface changes:

- **patch** — bug fixes, refactoring, small internal changes with no new user-facing behavior
- **minor** — new features or capabilities added in a backwards-compatible way
- **major** — breaking changes to the API, CLI, or behavior

For example, a new CLI flag is a minor bump to the app version, but only a patch bump to the Helm chart if the chart's own interface (values, templates) didn't change. If the chart gained a new configurable value, that's a minor bump to the chart as well.

✅ Good:
- Apply a semver minor bump to `internal/version/VERSION` and a patch bump to `charts/ralph-webhook/Chart.yaml`
- Apply a semver patch bump to `internal/version/VERSION` and `charts/ralph-webhook/Chart.yaml`

❌ Bad:
- Bump version to 3.2.11
- Set `appVersion` to "3.2.11" and `version` to "0.2.62"
