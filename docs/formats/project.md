# Project Format

Projects are YAML files that define work for the ralph agent to execute.

## File Location

Project files live at `./projects/<slug>.yaml`. The file name must match the project's `slug` field.

## Structure

```yaml
slug: project-identifier        # Used for branch naming (ralph/<slug>)
title: Brief description        # Used in PR title

feature: specs/features/<component>/<feature>   # Optional: link to feature directory

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
    passing: false
```

## Fields

### slug

Lowercase, hyphen-separated identifier. Becomes the branch name `ralph/<slug>`. Name the file to match: `<slug>.yaml`.

Examples: `user-authentication`, `fix-pagination`, `csv-export`

### title

Brief one-line description of what the project does. Used in the PR title.

### feature

Optional relative path to the feature directory under `specs/features/<component>/<feature>`. The directory contains `spec.md`, `orchestration.md`, and optionally `architecture.yaml`. Reference link for navigation — not injected into the agent's prompt.

### requirements

A list of one or more requirements. Each has:

- `slug` — lowercase, hyphen-separated identifier unique within the project. Used by ralph to track which requirement is being picked or updated.
- `description` — what the requirement covers
- `passing` — `false` = needs work (agent implements it), `true` = already done (agent skips)
- `items` (optional) — behavioral outcomes for work that falls outside the spec and orchestration; no architecture decisions
- `scenarios` (optional) — GWT scenarios copied from the spec document
- `code` (optional) — code the project should implement: modules, function signatures, struct names
- `tests` (optional) — specific tests the project should implement

At least one of `items`, `scenarios`, `code`, or `tests` must be present.

## Writing Requirements

The agent sees only the selected requirement and the project file — not the spec or orchestration content. Requirements must be self-contained.

Use `scenarios` for behavioral requirements from the spec, `code` for architecture from the orchestration document, and `tests` for specific tests that must be written. Use `items` only for work that falls outside the spec and orchestration — additional constraints, edge cases, or operational requirements. Items must not contain architecture decisions.

Each helper function called from a code entry's `body` must have its own requirement with a fully-specified `code` entry. Copy any spec scenarios that directly relate to the helper into the requirement's `scenarios`. Use `items` to fill any remaining gaps.

## Scenarios

Copied from the spec document. Each has a `title` and `items` (GWT steps).

## Code

Code entries describe the functions and shapes the project should implement, copied from the orchestration document.

All fields are required:

- `name` — the function or method name
- `description` — short summary of the entry's purpose
- `module` — the module where the code belongs, matching a `path` entry in the relevant architecture document
- `body` — the code to implement. Can be the full implementation or just an interface signature

## Tests

Test entries describe the specific tests the project should implement. Each test entry has the same shape as a `code` entry.

All fields are required:

- `name` — the test function or method name
- `description` — what behavior the test verifies
- `module` — the module where the test belongs, matching a `path` entry in the relevant architecture document
- `body` — the test code to implement. Can be the full implementation or just a signature with intent

## Version Bumps

If the repo uses versioning, include a `version` requirement. Specify the bump level — not the target version. Ralph determines the current version and applies the bump.

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
    passing: false

  - slug: build-csv-helper
    description: Build CSV bytes from report entries
    code:
      - name: buildCSV
        description: converts report entries to CSV bytes
        module: internal/reports
        body: |
          func buildCSV(entries []Entry) ([]byte, error)
    passing: false

  - slug: export-error-handling
    description: Export fails gracefully for invalid or missing reports
    items:
      - A request for a non-existent report ID returns 404
      - A malformed report ID returns 400 with a descriptive error message
    passing: false

  - slug: version-bump
    description: Version bump
    items:
      - Apply a semver minor bump to the app version
    passing: false
```

## Validation

Validate a project file with:

```sh
ralph validate ./projects/<slug>.yaml
```
