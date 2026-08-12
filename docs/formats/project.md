# Project Format

A project is any YAML or JSON file that contains an array of work items.

Ralph imposes no schema on that file. It needs exactly one thing from it: a **jq query** that resolves to an array. Each element of that array is an **item** — one unit of work, one iteration. Everything else in the file is opaque to ralph and is passed through to the AI agent as-is.

## File Location

Project files conventionally live at `./projects/<slug>.yaml`, but any path works. The extension determines the parser: `.yaml`/`.yml` or `.json`.

## Item Query

The item query is a [jq](https://jqlang.org/manual/) expression evaluated against the parsed file, using [gojq](https://github.com/itchyny/gojq). It defaults to `.`, so a file whose top level is an array needs no configuration at all.

```yaml
# projects/csv-export.yaml — top-level array, default query
- Add a CSV serializer for report entries
- Add GET /reports/:id/export returning text/csv
- Return 404 for unknown report IDs
```

Set `items` in [`.ralph/config.yaml`](../config.md) to pull the array out of a nested document:

```yaml
# .ralph/config.yaml
items: .requirements
```

```yaml
# projects/csv-export.yaml
slug: csv-export
title: Add CSV export to the reports API
requirements:
  - slug: csv-serializer
    description: Serialize report entries as CSV
  - slug: export-endpoint
    description: Expose the export over HTTP
```

Override per run with `--items`:

```bash
ralph projects/csv-export.yaml --items '.spec.tasks'
```

### Resolution rules

The query is evaluated and its outputs collected:

- **One output, and it is an array** — the array's elements are the items. `.requirements`
- **Any other case** — each output is an item. `.requirements[]` and `.backend[], .frontend[]` both work; a query returning a single scalar yields a single item.

Empty outputs are then dropped, so resolution produces either nothing at all or a list in which every item has content. An output is empty when it is null, `false`, `0`, a string that is empty or only whitespace, `{}`, or `[]`.

```yaml
requirements:
  - Add a CSV serializer for report entries
  -                                          # null — dropped
  - ""                                       # dropped
  - Add GET /reports/:id/export              # index 1, not 3
```

Dropping happens before indices are assigned, so an index is a position in the surviving list. Every command resolves the same way, so `ralph run`, `ralph get`, and `ralph validate` all agree on it.

When nothing survives — no output at all, or only empty outputs — the command that needs the items reports `item query yielded no items: <query>` and does no work. For a run that means it stops before the first iteration rather than opening a pull request on an empty project.

Because both `.requirements` and `.requirements[]` produce the same result, either form is fine. Prefer the array form.

### Choosing a query for foreign files

The point of the query is that a project does not have to be a ralph file. Any YAML or JSON document with a list of work in it can drive a run:

```yaml
items: .jobs                              # a CI config
items: '.issues | map(select(.state == "open"))'   # an exported issue list
items: '.tasks[] | select(.assignee == "ralph")'   # a task file, filtered
```

Filtering in the query is fine — the resolved array *is* the project as far as ralph is concerned, including for item indexing and completion tracking. The query is resolved once per run and stays fixed for that run's lifetime; see [Iterations](../iterations.md#the-project-file-is-immutable).

## Items

An item can be any YAML/JSON value: a string, a mapping, a nested structure. Ralph reads two things from it.

### Index

The item's 0-based position in the resolved array. Always present, and the only thing that identifies an item to ralph.

### Key

If the item is a mapping with a scalar `slug`, `id`, or `name` field — checked in that order — that value is the item's **key**.

```yaml
- slug: csv-serializer      # key: "csv-serializer"
- id: 4821                  # key: "4821"
- name: export-endpoint     # key: "export-endpoint"
- Add a CSV serializer      # no key
```

The key is a convenience: it labels the item in commit messages, logs, and picker output, so `Ralph item 0 (csv-serializer) completed` reads better than `Ralph item 0 completed` and greps better later. It is not an identifier — ralph tracks items by index. Keys need not be unique and nothing breaks if they are not.

## Optional Metadata

Two top-level fields are read when present, and only when the file's top level is a mapping. Both are optional.

| Field | Used for | Fallback |
|-------|----------|----------|
| `slug` | Branch name `ralph/<slug>` | The project file's base name |
| `title` | Pull request title | The slug |

A project file that is a top-level array has neither, so both derive from the file name. See [run/spec.md](../../specs/features/ralph/run/spec.md) for slug sanitization rules.

## No Completion State in the File

Items do not carry a `passing` field. Nothing writes progress back into the project file — not the AI agent, not ralph. Completion is recorded in the branch's commit messages instead:

```
feat: add CSV serializer for report entries

Ralph item 0 (csv-serializer) completed
```

Ralph reads `git log <base>..HEAD` at the start of every iteration to determine which items are done. The project file is read-only from the first iteration to the last. See [Iterations](../iterations.md#the-project-file-is-immutable).

To see that state for a given file, ask ralph:

```bash
ralph get complete                                 # [0, 2]
ralph get incomplete ./projects/<slug>.yaml        # the items still to do
```

## Conventional Item Shape

Ralph does not require this shape, but it is what [`ralph-write-project`](../../.claude/skills/ralph-write-project/SKILL.md) generates and what the default agent instructions are tuned for. Use it when the project is authored for ralph rather than borrowed from another tool.

```yaml
slug: project-identifier        # branch name (ralph/<slug>)
title: Brief description        # PR title

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

Paired with `items: .requirements` in config.

### Fields

- `slug` — lowercase, hyphen-separated label, conventionally unique within the project. Becomes the item key.
- `description` — what the requirement covers
- `items` (optional) — behavioral outcomes for work that falls outside the spec and orchestration; no architecture decisions
- `scenarios` (optional) — GWT scenarios copied from the spec document
- `code` (optional) — code the project should implement: modules, function signatures, struct names
- `tests` (optional) — specific tests the project should implement

At least one of `items`, `scenarios`, `code`, or `tests` should be present — an item with only a slug and description gives the agent nothing to build.

### Writing Items

The agent sees the selected item and the full project file — not the spec or orchestration content. Items must be self-contained.

Use `scenarios` for behavioral requirements from the spec, and `code` and `tests` for implementation and test shapes sourced directly from the orchestration document — never invented. Use `items` only for work that falls outside the spec and orchestration — additional constraints, edge cases, or operational requirements. Items must not contain architecture decisions.

Each helper function called from a code entry's `body` must have its own item with a fully-specified `code` entry. Copy any spec scenarios that directly relate to the helper into that item's `scenarios`. Use `items` to fill any remaining gaps.

See [Writing Good Requirements](../writing-requirements.md).

### Code and Tests

Code entries relay implementation shapes from the feature's orchestration document to the ralph agent. Every entry must be sourced directly from `orchestration.md` — not composed freehand. If the feature has no orchestration document, or the orchestration has no matching shape for an item, omit the `code` field entirely and use `scenarios` and `items` instead. Test entries follow the same rule for test shapes.

All fields are required in both:

- `name` — the function, method, or test name
- `description` — short summary of the entry's purpose
- `module` — the module where the code belongs, matching a `path` entry in the relevant architecture document
- `body` — the code to implement. Can be the full implementation or just an interface signature

### Version Bumps

If the repo uses versioning, include a version item. Specify the bump level — not the target version. Ralph determines the current version and applies the bump.

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

## Validation

```sh
ralph validate ./projects/<slug>.yaml
ralph validate ./projects/<slug>.yaml --items '.requirements'
```

Validation checks only what ralph depends on:

1. The file parses as YAML or JSON.
2. The item query evaluates without error.
3. The query resolves to at least one non-empty item.

There is no schema check — anything that parses and yields non-empty items is a valid project. When a file fails to parse, `ralph validate` still runs its bounded AI fix loop to repair the syntax, then rewrites the file in canonical YAML.

`--items` overrides the query for one invocation; otherwise validate uses `items` from `.ralph/config.yaml`, then `.` — the same resolution a run uses, so validate and run agree by default. Passing a query to one and not the other is the only way to get a file that validates but does not run.
