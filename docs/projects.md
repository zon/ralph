# Project Files

A project is any YAML or JSON file that contains an array of work items. Ralph imposes no schema on a project file. It needs exactly one thing from it: a [jq](https://jqlang.org/manual/) query that resolves to an array. Each element of that array is an **item**: one unit of work, one iteration. Everything else in the file is opaque to Ralph and is passed through to the AI agent as-is.

What a project file should contain beyond that is defined outside Ralph: the conventional shape of an item and the fields a well-formed project carries. See [Project Format](zpecs/project.md) in the installed spec documents.

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

Set `items` in [`.ralph/config.yaml`](../internal/config/config.md) to pull the array out of a nested document:

```yaml
# .ralph/config.yaml
items: .requirements
```

Override per run with `--items`:

```bash
ralph projects/csv-export.yaml --items '.spec.tasks'
```

### Resolution rules

The query is evaluated and its outputs collected:

- **One output, and it is an array** — the array's elements are the items. `.requirements`
- **Any other case** — each output is an item. `.requirements[]` and `.backend[], .frontend[]` both work. A query returning a single scalar yields a single item.

Empty outputs are then dropped, so resolution produces either nothing at all or a list in which every item has content. An output is empty when it is null, `false`, `0`, a string that is empty or only whitespace, `{}`, or `[]`.

```yaml
requirements:
  - Add a CSV serializer for report entries
  -                                          # null — dropped
  - ""                                       # dropped
  - Add GET /reports/:id/export              # index 1, not 3
```

Dropping happens before indices are assigned, so an index is a position in the surviving list. Every command resolves the same way, so `ralph run`, `ralph complete`, `ralph incomplete`, and `ralph validate` all agree on it.

When nothing survives (no output at all, or only empty outputs), the command that needs the items reports `item query yielded no items: <query>` and does no work. For a run that means it stops before the first iteration rather than opening a pull request on an empty project.

Because both `.requirements` and `.requirements[]` produce the same result, either form is fine. Prefer the array form.

### Choosing a query for foreign files

The point of the query is that a project does not have to be a Ralph file. Any YAML or JSON document with a list of work in it can drive a run:

```yaml
items: .jobs                              # a CI config
items: '.issues | map(select(.state == "open"))'   # an exported issue list
items: '.tasks[] | select(.assignee == "ralph")'   # a task file, filtered
```

Filtering in the query is fine. The resolved array is the project as far as Ralph is concerned, including for item hashing and completion tracking. The query is resolved once per run and stays fixed for that run's lifetime. See [Iterations](iterations.md#the-project-file-is-immutable).

## Item Hash

The 7-character base-62 hash of the item's text, computed as a SHA-256 digest of the text normalized by trimming surrounding whitespace and lower-casing. Always present, and the only thing that identifies an item to Ralph. The same text always yields the same hash, so an item keeps its completion across runs as long as its text is unchanged.

## Item Key

If the item is a mapping with a scalar `slug`, `id`, or `name` field, checked in that order, that value is the item's **key**.

```yaml
- slug: csv-serializer      # key: "csv-serializer"
- id: 4821                  # key: "4821"
- name: export-endpoint     # key: "export-endpoint"
- Add a CSV serializer      # no key
```

The key is a convenience: it labels the item in logs and picker output. It is not an identifier. Ralph tracks items by hash. Keys need not be unique and nothing breaks if they are not.

## Optional Metadata

Two top-level fields are read when present, and only when the file's top level is a mapping. Both are optional.

| Field | Used for | Fallback |
|-------|----------|----------|
| `slug` | Branch name `ralph/<slug>` | The project file's base name |
| `title` | Pull request title | The slug |

A project file that is a top-level array has neither, so both derive from the file name.

## No Completion State in the File

Items do not carry a completion field. Neither the AI agent nor Ralph writes progress back into the project file. Completion is recorded in the branch's commit messages instead, as a bare `<branch>-<hash>` trailer line:

```
feat: add CSV serializer for report entries

csv-export-IYAWN02
```

Ralph reads `git log <base>..HEAD` at the start of every iteration to determine which items are done. The project file is read-only from the first iteration to the last. See [Iterations](iterations.md#the-project-file-is-immutable).

To see that state for a given file, ask Ralph:

```bash
ralph complete                                   # completion hashes, one per line
ralph incomplete ./projects/<slug>.yaml          # the items still to do
```

## Validation

```sh
ralph validate ./projects/<slug>.yaml
ralph validate ./projects/<slug>.yaml --items '.requirements'
```

Validation checks only what Ralph depends on:

1. The file parses as YAML or JSON.
2. The item query evaluates without error.
3. The query resolves to at least one non-empty item.

There is no schema check: anything that parses and yields non-empty items is a valid project. When a file fails to parse, `ralph validate` still runs its bounded AI fix loop to repair the syntax, then rewrites the file in canonical YAML.
