# Project Format

A project is a YAML or JSON file containing an array of work items. Each element of that array is an **item** — one unit of work, one iteration of a coding agent.

This document defines the **conventional shape** for a project authored against a spec, an architecture, and an orchestration. It is a convention, not a runner requirement: a runner needs only an array of items and does not impose a schema. What follows is the shape that carries enough context for an agent to implement an item without re-reading the source documents.

## File Location

Project files live at `./projects/<slug>.yaml`. See [Directory Structure](README.md#directory-structure).

## Shape

```yaml
slug: project-identifier        # branch name
title: Brief description        # pull request title

feature: specs/features/<component>/<feature>   # optional: link to feature directory

requirements:
  - slug: requirement-identifier
    description: What should happen
    items:
      - Specific behavioral outcome the agent must achieve
    scenarios:
      - title: Scenario title
        items:
          - GIVEN ...
          - WHEN ...
          - THEN ...
    code:
      - name: ExampleFunc
        description: optional summary of what this function does
        module: path/to/module
        body: |
          func ExampleFunc() {
            // target implementation shape
          }
    tests:
      - name: TestExampleFunc
        description: verifies ExampleFunc handles the happy path
        module: path/to/module
        body: |
          func TestExampleFunc(t *testing.T) {
            // assertions
          }
```

The item array is nested under `requirements`, so the runner's item query is `.requirements`. A top-level array works too and needs no query, but loses `slug` and `title`.

### Top-level fields

| Field | Used for |
|-------|----------|
| `slug` | Branch name. Falls back to the file's base name. |
| `title` | Pull request title. Falls back to the slug. |
| `feature` | Optional path to the feature directory the project implements. |

### Item fields

- `slug` — lowercase, hyphen-separated label, conventionally unique within the project. Runners use `slug`, `id`, or `name` to label the item in commit messages and logs, so give every item one.
- `description` — what the requirement covers
- `items` (optional) — behavioral outcomes for work that falls outside the spec and orchestration; no architecture decisions
- `scenarios` (optional) — GWT scenarios copied from the spec document
- `code` (optional) — code the project should implement: modules, function signatures, struct names
- `tests` (optional) — specific tests the project should implement

At least one of `items`, `scenarios`, `code`, or `tests` must be present — an item with only a slug and description gives the agent nothing to build.

## No Completion State in the File

Items do not carry a `passing`, `done`, or `complete` field, and nothing writes progress back into the project file. Completion belongs to the runner's own record — conventionally the branch's commit history — which keeps the project file read-only from the first iteration to the last and means a half-written file can never corrupt a run.

## Writing Items

The agent sees the selected item and the full project file — not the spec or orchestration content. Items must be self-contained.

Use `scenarios` for behavioral requirements from the spec, and `code` and `tests` for implementation and test shapes sourced directly from the orchestration document — never invented. Use `items` only for work that falls outside the spec and orchestration — additional constraints, edge cases, or operational requirements. Items must not contain architecture decisions.

Each helper function called from a code entry's `body` must have its own item with a fully-specified `code` entry. Copy any spec scenarios that directly relate to the helper into that item's `scenarios`. Use `items` to fill any remaining gaps.

One item is one iteration. Work that needs three separate rounds should be three items. See [Writing Requirements](requirements.md).

## Code and Tests

Code entries relay implementation shapes from the feature's [orchestration document](orchestration.md) to the agent. Every entry must be sourced directly from `orchestration.md` — not composed freehand. If the feature has no orchestration document, or the orchestration has no matching shape for an item, omit the `code` field entirely and use `scenarios` and `items` instead. Test entries follow the same rule for test shapes.

All fields are required in both:

- `name` — the function, method, or test name
- `description` — short summary of the entry's purpose
- `module` — the module where the code belongs, matching a `path` entry in the relevant [architecture document](architecture.md)
- `body` — the code to implement. Can be the full implementation or just an interface signature

The module's `category` in the architecture document — and that category's `signatures` and `orchestration` flag — determine what form the code and tests may take. An orchestration module gets domain logic and orchestration tests; an implementation module gets real implementations, mocks, and unit tests. See [Writing Code](code.md).

## Version Bumps

If the repo uses versioning, include a version item. Specify the bump level — not the target version. The agent determines the current version and applies the bump.

Each versioned resource is bumped independently based on how its own interface changes:

- **patch** — bug fixes, refactoring, small internal changes
- **minor** — new features added in a backwards-compatible way
- **major** — breaking changes to the API, CLI, or behavior

## Example

```yaml
slug: csv-export
title: Add CSV export to the reports API

feature: specs/features/reports/csv-export

requirements:
  - slug: export-report-endpoint
    description: Reports can be exported as CSV files
    scenarios:
      - title: Successful CSV export
        items:
          - GIVEN a report with three entries
          - WHEN GET /reports/:id/export is called
          - THEN the response has Content-Type text/csv and three data rows
    code:
      - name: ExportReport
        description: handler that exports a report as CSV
        module: internal/reports
        body: |
          func ExportReport(id string) ([]byte, error)
    tests:
      - name: TestExportReport_Success
        description: verifies a report with entries exports as CSV with the correct content type
        module: internal/reports
        body: |
          func TestExportReport_Success(t *testing.T)

  - slug: build-csv-helper
    description: Build CSV bytes from report entries
    code:
      - name: buildCSV
        description: converts report entries to CSV bytes
        module: internal/reports
        body: |
          func buildCSV(entries []Entry) ([]byte, error)

  - slug: export-error-handling
    description: Export fails gracefully for invalid or missing reports
    items:
      - A request for a non-existent report ID returns 404
      - A malformed report ID returns 400 with a descriptive error message

  - slug: version-bump
    description: Version bump
    items:
      - Apply a semver minor bump to the app version
```
