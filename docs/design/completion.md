# Completion Design

Ralph records progress in commit messages, not in the project file. The set of completion trailers on a branch is the only record of what a project has finished.

## Completion Hash

A completion hash identifies an item by its text. Ralph hashes the item's canonical YAML text after trimming surrounding whitespace and lower-casing it with SHA-256, keeps the first 40 bits, and encodes them in base 62 to 7 characters.

The normalization means two items that differ only in case or surrounding whitespace share a hash. Any other text change makes a new item.

## Completion Trailer

A completion trailer is the last line of an iteration's commit message:

```
<branch>-<hash>
```

For example, `csv-export-IYAWN02`. The branch is the project branch the trailer belongs to. The hash is the item's completion hash.

Rules:

* Only a bare trailer line counts. Prose around a hash does not.
* Only trailers naming the current branch count. A trailer from a parent project branch is ignored without a warning.
* A commit may carry more than one trailer.
* When a project is resolved, a trailer hash that matches no item is dropped with a warning.

The development agent writes the trailer at the end of `report.md` when the item is done. Ralph commits the report verbatim and never adds or removes a trailer. See [Run Design](run.md).

## Reading Completion

Ralph reads the commit messages on the current branch that are not on the base branch. For `ralph complete`, the base branch is `--base`, else `defaultBranch` from the config. A run uses the base branch it resolved. This bounds the history a project inherits from the branch it started on.

Completion is read at the start of every iteration. When the run exhausts its iteration limit, Ralph reads it again to name the items still incomplete. The list of incomplete items is what the picker chooses from.

## Pick Reference

The pick reference is the completion hash shown to the picker and reported back:

```
ref: <hash>
```

The picker writes it to `picked-item.txt`. A reference names the item without asking the agent to reproduce text, whose YAML quoting and escaping can change. Ralph also accepts a reported item text or its canonical YAML, so a picker that copies text still names its selection.

## Commands

`ralph complete [project]` reads the branch's trailers and prints the hashes, sorted and deduplicated, one per line. It accepts `--json` for a JSON array, and `--base` to bound the log. The project file is optional. Without it, Ralph reports every trailer hash on the branch without checking it against a resolved item, so a branch whose project file was cleaned up still reports its completion.

`ralph incomplete <project>` resolves the items, removes the complete ones, and prints the rest as a JSON array in array order.

Both commands are read-only. They do not invoke an agent, write a file, or switch a branch.
