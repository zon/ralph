# Project Files

A project is a YAML or JSON file listing the work to do. Ralph imposes no schema on the file. It needs one thing from it: an array of items, selected with a jq query. Each item is one unit of work, one iteration. Everything else in the file is opaque and passes through to the AI agent as-is.

The conventional item shape, what fields a well-formed project carries, is defined outside Ralph. Ralph runs whatever it is given.

## A Simple Project

The simplest project is a list of items in `projects/`, named `<slug>.yaml`. The slug names the branch and the pull request:

```yaml
- Reports can be exported as CSV from GET /reports/:id/export
- A request for a missing report ID returns 404
- A malformed report ID returns 400 with an error message
```

Run the project by path. The extension picks the parser: `.yaml` or `.yml` for YAML, `.json` for JSON:

```bash
ralph projects/report-export.yaml
```

## What Ralph Does

Each list entry is one item. Ralph creates a branch named after the file, `report-export`, and implements the items, one per iteration. A finished item is marked by a completion trailer in its commit message, so completion lives in the branch history and the file is never written back. When every item is complete, Ralph opens a pull request.

## Item Query

Ralph resolves a file to its items with a [jq](https://jqlang.org/manual/) query. The default query is `.`, so a file whose top level is already a list needs no configuration. Point the query at a nested list with the `items` option in `.ralph/config.yaml` (see `ralph help config`), or override it per run with `--items`:

```yaml
# .ralph/config.yaml
items: .requirements
```

```bash
ralph projects/report-export.yaml --items '.spec.tasks'
```

The point of the query is that the project does not have to be a Ralph file. Any YAML or JSON document with a list of work in it can drive a run, and filtering in the query is fine:

```yaml
items: .jobs                                  # a CI config
items: '.issues | map(select(.state == "open"))'   # an exported issue list
items: '.tasks[] | select(.assignee == "ralph")'   # a task file, filtered
```

### Resolution rules

The query is evaluated and its outputs are collected:

- One output that is an array: the array's elements are the items. `.requirements` selects a nested list this way.
- Any other case: each output is an item. `.requirements[]` and `.backend[], .frontend[]` both work. A query returning a single scalar yields a single item.

`.requirements` and `.requirements[]` resolve to the same items, so either form is fine. Prefer the array form.

Dropping happens before indices are assigned, so an index is a position in the surviving list. An output is empty when it is null, `false`, `0`, a string that is empty or only whitespace, `{}`, or `[]`:

```yaml
requirements:
  - Add a CSV serializer for report entries
  -                                          # null — dropped
  - ""                                       # dropped
  - Add GET /reports/:id/export              # index 1, not 3
```

When nothing survives, the command that needs the items reports `item query yielded no items: <query>` and does no work. A run stops before the first iteration instead of opening a pull request on an empty project.

The resolved array is the project as far as Ralph is concerned, including for item hashing and completion tracking. It is resolved once per run and stays fixed for the run's lifetime. Every command resolves items the same way, so `ralph run`, `ralph complete`, `ralph incomplete`, and `ralph validate` agree on them.

## Item Hash

An item's hash is a 7-character base-62 encoding of a SHA-256 digest of its text, normalized by trimming surrounding whitespace and lower-casing. It is always present, and it is the only thing that identifies an item to Ralph. The same text always yields the same hash, so an item keeps its completion across runs as long as its text is unchanged.

## Item Key

A mapping item with a scalar `slug`, `id`, or `name` field, checked in that order, has that value as its key:

```yaml
- slug: csv-serializer      # key: "csv-serializer"
- id: 4821                  # key: "4821"
- name: export-endpoint     # key: "export-endpoint"
- Add a CSV serializer      # no key
```

The key is a convenience that labels the item in logs and picker output. It is not an identifier. Ralph tracks items by hash, keys need not be unique, and nothing breaks if they are not.

## Project Metadata

Two optional fields are read when present, and only when the file's top level is a mapping:

| Field | Used for | Fallback |
|-------|----------|----------|
| `slug` | Branch name `<slug>` | The file's base name |
| `title` | Pull request title | The slug |

A project file whose top level is an array has neither, so both come from the file name.

## Where Completion Lives

Items carry no completion field, and neither the AI agent nor Ralph writes progress back into the file. Completion is recorded in the branch's commit messages as a bare `<branch>-<hash>` trailer line:

```
feat: export reports as CSV

report-export-IYAWN02
```

Ralph reads `git log <base>..HEAD` at the start of every iteration to see which items are done. Because the file is not the record, you can edit it during a run: Ralph re-evaluates any item whose text you changed. When a project file has changed shape, start a fresh branch.

To see that state for a given file, ask Ralph:

```bash
ralph complete                                   # completion hashes on this branch, one per line
ralph incomplete ./projects/<slug>.yaml          # the items still to do
```

## Check the Project

`ralph validate` parses a project file and checks that its item query resolves to at least one non-empty item:

```bash
ralph validate projects/report-export.yaml
ralph validate projects/report-export.yaml --items '.requirements'
```

There is no schema check: anything that parses and yields non-empty items is a valid project. Validation checks only what a run depends on: the file parses as YAML or JSON, the item query evaluates without error, and it resolves to at least one non-empty item. When a file fails to parse, `ralph validate` runs a bounded AI fix loop to repair the syntax, then rewrites the file in canonical YAML.
