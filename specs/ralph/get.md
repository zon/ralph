# Get Command Specification

## Purpose

Define the behavior of `ralph get complete` and `ralph get incomplete`, the read-only primitives that report which items of a project are done and which are left. Completion is recorded in the commit messages on the project branch, not in the project file, so these commands read the branch's commit log and the resolved item array and report the difference. Both are the same operations the iteration loop performs, exposed so a run can be inspected from a script or by hand.

Neither command invokes an AI agent and neither writes to the repository.

## Requirements

### Requirement: Command Invocation

The system SHALL provide a `ralph get` command with two subcommands: `complete` and `incomplete`. `complete` accepts an optional project file path. `incomplete` requires a project file path.

#### Scenario: Complete invoked without a project file

- GIVEN the user runs `ralph get complete`
- WHEN the command starts
- THEN the completion trailers are read from the commit log with no item array resolved

#### Scenario: Complete invoked with a project file

- GIVEN the user runs `ralph get complete ./projects/csv-export.yaml`
- WHEN the command starts
- THEN the item array is resolved from that file and used to bound the reported hashes

#### Scenario: Incomplete invoked without a project file

- GIVEN the user runs `ralph get incomplete` with no positional argument
- WHEN the command starts
- THEN the command exits with a non-zero status and reports that a project file path is required

#### Scenario: Project file not found

- GIVEN a project file path that does not exist on disk
- WHEN either subcommand runs with that path
- THEN the command exits with a non-zero status and reports that the file was not found

---

### Requirement: Item Query Resolution

Both subcommands SHALL resolve the item array from the project file using a jq query resolved with two-level precedence: `--items` at the command line takes priority. Otherwise the `items` field in `.ralph/config.yaml` is used. Otherwise the query defaults to `.`. This is the same resolution the run command and `ralph validate` use, so all three agree on what the items are, and on their indices, by default.

Resolution discards empty outputs, so the resolved array is either empty or made entirely of non-empty items. See [the project file format](../../docs/projects.md#item-query). An empty resolved array SHALL be reported as an error, because there are no items to report on.

#### Scenario: `--items` flag takes precedence

- GIVEN `items: .requirements` is set in `.ralph/config.yaml`
- AND the user passes `--items '.spec.tasks'`
- WHEN the item array is resolved
- THEN `.spec.tasks` is evaluated against the project file

#### Scenario: Config query used when no flag is passed

- GIVEN `items: .requirements` is set in `.ralph/config.yaml`
- AND no `--items` flag is passed
- WHEN the item array is resolved
- THEN `.requirements` is evaluated against the project file

#### Scenario: Default query when flag and config are unset

- GIVEN `items` is not set in `.ralph/config.yaml`
- AND no `--items` flag is passed
- WHEN the item array is resolved
- THEN the query `.` is evaluated, so a file whose top level is an array resolves to that array

#### Scenario: Query yields no items

- GIVEN a project file and a query that produces no output
- WHEN the item array is resolved
- THEN an error is returned: `item query yielded no items: <query>`

#### Scenario: Query yields only empty items

- GIVEN a project file whose item list holds nothing but nulls, blank strings, and empty mappings
- WHEN the item array is resolved
- THEN the resolved array is empty and an error is returned: `item query yielded no items: <query>`

#### Scenario: Empty entries dropped before indexing

- GIVEN a project file whose item list holds two work items with a null entry between them
- WHEN the item array is resolved
- THEN two items are reported, indexed 0 and 1
- AND completion is matched against those items' hashes, the same ones a run records

---

### Requirement: Base Branch Resolution

Both subcommands SHALL bound the commit log by a base branch, resolved from `--base` (`-B`) when provided and otherwise from the `defaultBranch` field in `.ralph/config.yaml`. The commits considered are those on the current branch that are not on the base branch.

#### Scenario: `--base` overrides the configured default branch

- GIVEN `defaultBranch: main` is set in `.ralph/config.yaml`
- AND the user passes `--base develop`
- WHEN the commit log is read
- THEN only commits on the current branch that are not on `develop` are considered

#### Scenario: Configured default branch used when no flag is passed

- GIVEN `defaultBranch: main` is set in `.ralph/config.yaml`
- AND no `--base` flag is passed
- WHEN the commit log is read
- THEN only commits on the current branch that are not on `main` are considered

---

### Requirement: Completion Trailer Parsing

The system SHALL treat a commit as recording an item complete when its message contains a trailer of the form `<branch>-<hash>`, where `<branch>` is the project branch the trailer belongs to and `<hash>` is a 7-character base-62 encoding of a SHA-256 digest of the item's text, normalized by trimming surrounding whitespace and lower-casing. A single commit MAY carry more than one trailer. The hash alone identifies the item. Only trailers whose branch matches the current branch SHALL be honored. A trailer from any other branch is ignored without a warning, so a project branched from another project's branch never inherits that project's completion.

#### Scenario: Trailer on the current branch recognized

- GIVEN a commit whose message ends with `csv-export-IXBRf1x`
- AND the hash `IXBRf1x` matches the text of a resolved item
- AND the current branch is `csv-export`
- WHEN completion is read
- THEN that item is reported complete

#### Scenario: Trailer from another branch ignored

- GIVEN a commit whose message ends with `feature-a-D3XocZs`
- AND the current branch is `csv-export`
- WHEN completion is read
- THEN no item is reported complete, because the trailer names a different project branch

#### Scenario: Multiple trailers in one commit

- GIVEN a commit whose message ends with a paragraph containing `csv-export-D3XocZs` and `csv-export-9d8LxCD`
- AND those hashes match the texts of two resolved items
- WHEN completion is read
- THEN both of those items are reported complete

#### Scenario: Commit without a trailer completes nothing

- GIVEN a commit whose message contains no completion trailer
- WHEN completion is read
- THEN that commit contributes no completed hashes

#### Scenario: Unmatched hash ignored with a warning

- GIVEN a trailer `csv-export-BEAT2F1` whose hash matches no resolved item
- AND a project file was provided
- WHEN completion is read
- THEN a warning is emitted naming the unmatched hash
- AND that hash is not reported complete

---

### Requirement: Complete Output

`ralph get complete` SHALL print the completed item hashes to stdout as a JSON array, sorted and deduplicated, and exit with status 0.

#### Scenario: Completed hashes printed

- GIVEN the branch's commit log records items 2, 0, and 3 complete
- WHEN `ralph get complete` runs
- THEN the three item hashes are printed to stdout as a JSON array
- AND the command exits with status 0

#### Scenario: Duplicate trailers collapse

- GIVEN two commits on the branch both record the same item complete
- WHEN `ralph get complete` runs
- THEN that item's hash appears exactly once in the output

#### Scenario: Nothing complete

- GIVEN no commit on the branch carries a completion trailer
- WHEN `ralph get complete` runs
- THEN `[]` is printed to stdout
- AND the command exits with status 0

#### Scenario: Unbounded output without a project file

- GIVEN no project file is provided
- AND the log contains a trailer `csv-export-BEAT2F1` on the current branch `csv-export`
- WHEN `ralph get complete` runs
- THEN the hash `BEAT2F1` is reported without a resolved item check, because no item array was resolved

#### Scenario: Works after the project file is removed

- GIVEN the project file was deleted by a cleanup commit on this branch
- WHEN `ralph get complete` runs without a project file argument
- THEN the completed hashes are still reported from the commit log

---

### Requirement: Incomplete Output

`ralph get incomplete` SHALL resolve the item array, remove the items reported complete, and print the remaining items to stdout as a JSON array in their original array order.

#### Scenario: Remaining items printed

- GIVEN a project resolving to 4 items
- AND items 0 and 2 are recorded complete
- WHEN `ralph get incomplete <file>` runs
- THEN the items at indices 1 and 3 are printed as a JSON array, in that order
- AND each item is printed exactly as it appears in the resolved array

#### Scenario: Every item complete

- GIVEN every resolved item is recorded complete
- WHEN `ralph get incomplete <file>` runs
- THEN `[]` is printed to stdout
- AND the command exits with status 0

---

### Requirement: Incomplete Index Output

`ralph get incomplete` SHALL accept an `--index` flag that emits the indices of the incomplete items rather than the items themselves, in the same JSON array form.

#### Scenario: Indices emitted instead of items

- GIVEN a project resolving to 5 items
- AND items 0, 2, and 3 are recorded complete
- WHEN `ralph get incomplete <file> --index` runs
- THEN `[1, 4]` is printed to stdout

#### Scenario: `--index` rejected on complete

- GIVEN the user runs `ralph get complete --index`
- WHEN the command validates its flags
- THEN an error is returned, because `complete` already emits hashes

---

### Requirement: Read-Only Execution

Both subcommands SHALL be read-only and SHALL NOT invoke an AI agent. No file is written, no commit is created, and no branch is switched.

#### Scenario: No AI invocation

- GIVEN either subcommand runs
- WHEN it resolves items and reads the commit log
- THEN no AI agent is invoked

#### Scenario: Repository left untouched

- GIVEN either subcommand runs against a repository with a clean working tree
- WHEN the command finishes
- THEN the working tree is still clean and the current branch is unchanged
- AND the project file is byte-identical to what it was before the command ran
