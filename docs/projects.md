# Writing Projects

Projects are YAML files that define work for AI agents.

## Format

```yaml
name: project-identifier          # Used for branch naming (ralph/<name>)
description: Brief description    # Used in PR title

spec: specs/features/<area>/<feature>/spec.md   # Optional: link to feature spec
flow: specs/features/<area>/<feature>/flow.md   # Optional: link to implementation flow

requirements:
  - description: What should happen
    items:
      - Specific behavioral outcome the coder must achieve
      - Another outcome; see [flow functionName](../specs/features/<area>/<feature>/flow.md) for target implementation shape
    passing: false                # false = needs work, true = complete
```

## Spec and Flow Fields

`spec` and `flow` are reference links to the feature's documentation. They are not injected into the agent's prompt — they exist so the project is navigable alongside the specs and to indicate which feature this project implements.

## Writing Requirements

Requirements are the primary vehicle for information. They must contain everything the coder needs to implement the work. The agent sees only the selected requirement and the project file — not the spec or flow content.

Write each requirement as a set of specific behavioral outcomes. When the target implementation shape or acceptance criteria come from the spec or flow, copy the relevant detail directly into the requirement or link to the exact section:

- Copy key scenarios or constraints from the spec inline
- Reference the flow's function signatures or module boundaries when the coder should follow them
- Link to a specific function with a relative link like `[flow functionName](../specs/features/<area>/<feature>/flow.md)` or `[test functionName](../specs/features/<area>/<feature>/flow.md)` when the full context is too long to inline; spec sections use GitHub-style anchors like `[spec Requirement Name](../specs/features/<area>/<feature>/spec.md#requirement-name)`

Do not write architectural decisions into requirements (no package names, struct names, or implementation strategies). The agent decides how to structure code, guided by its instructions.

**When the flow has a `## Tests` section, include a requirement that links to it.** The agent does not see the flow document — it only sees requirements. Without an explicit link, the agent will write tests from scratch and will likely produce lower-level code that breaks the abstraction the flow defines. Add a requirement item like:

```
- Implement the tests from [flow ## Tests](../specs/features/<area>/<feature>/flow.md#tests) as written; each helper listed in `### Helpers` must be a real function encapsulating all infrastructure concerns
```

See [Writing Good Requirements](writing-requirements.md) for general guidance.

## Naming Projects

The `name` field becomes the branch name: `ralph/<name>`. Use lowercase, hyphen-separated identifiers:

- `user-authentication`
- `fix-pagination`
- `csv-export`

Name your project file to match: `user-authentication.yaml`.

Project files are stored in the `./projects` directory of the repo.

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
- Apply a semver minor bump to the app version file and a patch bump to the chart version
- Apply a semver patch bump to all versioned resources

❌ Bad:
- Bump version to 3.2.11
- Set appVersion to "3.2.11"

## Example

```yaml
name: csv-export
description: Add CSV export to the reports API

spec: specs/features/reports/csv-export/spec.md
flow: specs/features/reports/csv-export/flow.md

requirements:
  - description: Reports can be exported as CSV files
    items:
      - GET /reports/:id/export returns a CSV file with correct headers and MIME type
      - Rows map one-to-one with report entries; all fields are included
      - See [flow exportReport](../specs/features/reports/csv-export/flow.md) for target function shape
    passing: false

  - description: Export fails gracefully for invalid or missing reports
    items:
      - A request for a non-existent report ID returns 404
      - A malformed report ID returns 400 with a descriptive error message
    passing: false

  - category: versioning
    description: Version bump
    items:
      - Apply a semver minor bump to the app version
    passing: false
```
