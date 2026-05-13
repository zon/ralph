---
name: ralph-write-project
description: Creates and validates a ralph project YAML file defining work for the ralph agent to execute
---

# Ralph Project Format

Projects are YAML files that define work for the ralph agent.

## Format

```yaml
slug: project-identifier        # Used for branch naming (ralph/<slug>)
title: Brief description        # Used in PR title

spec: specs/features/<area>/<feature>/spec.md   # Optional: link to feature spec
flow: specs/features/<area>/<feature>/flow.md   # Optional: link to implementation flow

requirements:
  - description: What should happen
    items:
      - Specific behavioral outcome the agent must achieve
    scenarios:
      - title: Scenario title
        items:
          - GIVEN ...
          - WHEN ...
          - THEN ...
    flows:
      - name: ExampleFunc
        description: optional summary of what this flow does
        module: path/to/module.go
        code: |
          func ExampleFunc() {
            // target implementation shape
          }
        helpers:
          - name: helperFunc
            module: path/to/helpers.go
            description: what this helper does
    passing: false
```

## Fields

### slug

Lowercase, hyphen-separated identifier. Becomes the branch name `ralph/<slug>`. Name the file to match: `<slug>.yaml`.

Examples: `user-authentication`, `fix-pagination`, `csv-export`

### title

Brief one-line description of what the project does. Used in the PR title.

### spec and flow

Optional relative paths to the feature spec and flow documents. Reference links for navigation — not injected into the agent's prompt.

### requirements

A list of one or more requirements. Each has:

- `description` — what the requirement covers
- `passing` — `false` = needs work (agent implements it), `true` = already done (agent skips)
- `items` (optional) — behavioral outcomes for work that falls outside the spec and flow; no architecture decisions
- `scenarios` (optional) — GWT scenarios copied from the spec document
- `flows` (optional) — implementation shapes specifying the architecture: modules, function signatures, struct names, helpers

At least one of `items`, `scenarios`, or `flows` must be present.

## Writing Requirements

The agent sees only the selected requirement and the project file — not the spec or flow content. Requirements must be self-contained.

Use `scenarios` for behavioral requirements from the spec and `flows` for architecture from the flow document. Use `items` only for work that falls outside both — additional constraints, edge cases, or operational requirements not captured in the spec or flow. Items must not contain architecture decisions.

Each helper function listed in a flow's `helpers` must have its own requirement. The helper requirement includes a `flows` entry with `name` and optionally `module` and `description` — but no `code` or `helpers`. Copy any spec scenarios that directly relate to the helper into the requirement's `scenarios`. Use `items` to fill any remaining gaps.

## Scenarios

Copied from the spec document. Each has a `title` and `items` (GWT steps).

## Flows

Copied from the flow document. Each has:

- `name` — the method name (required)
- `description` — short summary of the flow's purpose (optional)
- `module` — where to write the flow (optional)
- `code` — the target implementation shape including signatures and strategies (optional)
- `helpers` — list of `{ name, module, description }` entries (optional)

Helper requirements use a flow entry with only `name`, `module`, and `description` — never `code` or `helpers`.

## Version Bumps

If the repo uses versioning, include a `version` requirement. Specify the bump level — not the target version. Ralph determines the current version and applies the bump.

Each versioned resource is bumped independently based on how its own interface changes:

- **patch** — bug fixes, refactoring, small internal changes
- **minor** — new features added in a backwards-compatible way
- **major** — breaking changes to the API, CLI, or behavior

## Steps

1. **Understand the work.** If the request is vague, ask clarifying questions before proceeding.

2. **Locate the spec and flow files** if the work targets a documented feature. Read them to extract requirements, scenarios, and flow shapes.

3. **Draft the requirements.** Copy relevant scenarios from the spec into `scenarios`. Copy relevant flow shapes into `flows`. For each helper listed in a flow's `helpers`, create a separate requirement with a flow entry containing `name` and optionally `module` and `description` (no `code` or `helpers`), relevant spec scenarios, and `items` to fill any gaps. Requirements must be self-contained — the agent sees only what is in the requirement.

4. **Write the file** to `./projects/<slug>.yaml` with all `passing: false`.

5. **Validate:**
   ```sh
   ralph validate ./projects/<slug>.yaml
   ```

6. **Report** the file path and a one-line summary.

## Example

```yaml
slug: csv-export
title: Add CSV export to the reports API

spec: specs/features/reports/csv-export/spec.md
flow: specs/features/reports/csv-export/flow.md

requirements:
  - description: Reports can be exported as CSV files
    scenarios:
      - title: Successful CSV export
        items:
          - GIVEN a report with three entries
          - WHEN GET /reports/:id/export is called
          - THEN the response has Content-Type text/csv and three data rows
    flows:
      - name: ExportReport
        module: internal/reports/export.go
        code: |
          func ExportReport(id string) ([]byte, error)
        helpers:
          - name: buildCSV
            module: internal/reports/csv.go
            description: converts report entries to CSV bytes
    passing: false

  - description: Build CSV bytes from report entries
    flows:
      - name: buildCSV
        module: internal/reports/csv.go
        description: converts report entries to CSV bytes
    passing: false

  - description: Export fails gracefully for invalid or missing reports
    items:
      - A request for a non-existent report ID returns 404
      - A malformed report ID returns 400 with a descriptive error message
    passing: false

  - description: Version bump
    items:
      - Apply a semver minor bump to the app version
    passing: false
```
