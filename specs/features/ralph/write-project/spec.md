# Project Format Specification

## Purpose

Define the format of ralph project files and the rules for writing them, and specify that the documentation describing this format is written as a Claude Code and opencode skill.

A project is any YAML or JSON file containing an array of work items. Ralph imposes no schema on it: it resolves the array with the [item query](../../../../docs/formats/project.md#item-query) and treats everything else in the file as opaque context passed through to the AI agent. The conventional item shape below is what the skill generates and what the default agent instructions are tuned for — a convention, not a requirement.

## Requirements

### Requirement: Project File

A project file MUST be a YAML or JSON document from which the item query resolves at least one non-empty item. No field is required and no field is rejected.

#### Scenario: Top-level array

- GIVEN a file whose top level is an array of work items
- WHEN it is used as a project with the default query
- THEN each element of the array is an item and the file is valid

#### Scenario: Nested array

- GIVEN a file whose top level is a mapping with a `requirements` list
- WHEN it is used as a project with the query `.requirements`
- THEN each element of that list is an item and the file is valid

#### Scenario: Foreign file used as a project

- GIVEN a YAML or JSON file authored by another tool, containing a list of work
- WHEN a query is chosen that resolves that list
- THEN the file is a valid project and ralph does not require any ralph-specific field in it

#### Scenario: Query resolves nothing

- GIVEN a file against which the item query produces no output
- WHEN the project is used
- THEN the resolved list is empty
- AND an error is reported naming the query, because there is no work to iterate over

### Requirement: Empty Items Are Dropped

Item query resolution MUST discard every empty output, so it produces either an empty list or a list in which every item is non-empty. An output is empty when it is null, `false`, the number zero, a string that is empty or contains only whitespace, an empty mapping, or an empty sequence. Every other value — including a mapping or sequence with any content at all — is an item.

Resolution itself MUST NOT fail when nothing survives; it returns the empty list. Reporting that as an error is the job of the command that needs work to do, and each one reports `item query yielded no items: <query>`; see [validate/spec.md](../validate/spec.md), [run-local/spec.md](../run-local/spec.md), and [get/spec.md](../get/spec.md).

Discarding MUST happen during resolution, before any index is assigned, so an item's index is its position in the surviving list. Because every command resolves the same query against the same file the same way, all of them agree on that index.

#### Scenario: Empty entries discarded

- GIVEN a list holding a work item, a null entry, an empty string, and a second work item
- WHEN the item query resolves
- THEN two items are resolved
- AND their indices are 0 and 1, as though the empty entries were never in the file

#### Scenario: Falsy scalars are empty

- GIVEN a list holding entries `false`, `0`, and `"   "`
- WHEN the item query resolves
- THEN none of them is an item

#### Scenario: Empty mappings and sequences are empty

- GIVEN a list holding an entry `{}` and an entry `[]`
- WHEN the item query resolves
- THEN neither is an item, because neither carries any work to do

#### Scenario: Query resolves only empty outputs

- GIVEN a file whose item list holds nothing but nulls and empty mappings
- WHEN the item query resolves
- THEN the resolved list is empty and no error comes from resolution itself
- AND the command using the project reports `item query yielded no items: <query>` and does no work

#### Scenario: Non-empty items are untouched

- GIVEN a list whose entries are plain strings, mappings, and nested structures, none of them empty
- WHEN the item query resolves
- THEN every entry is an item, in file order, and nothing is dropped

### Requirement: Item Index

Each item MUST be identified by its 0-based position in the resolved array. The index is the only identifier ralph uses. Because the project file is never written to during a run, an item's index refers to the same work for the whole run.

#### Scenario: Index identifies the item

- GIVEN a project resolving to four items
- WHEN the third element is selected for an iteration
- THEN it is identified as index 2 in the picker prompt, the development prompt, and the completion trailer

#### Scenario: Index is not written into the file

- GIVEN any project file
- WHEN it is authored
- THEN no index field is written into the items, because the index is the array position

### Requirement: Item Key

When an item is a mapping with a scalar `slug`, `id`, or `name` field — checked in that order — that value MUST be used as the item's key. The key is a label used in commit messages, logs, and picker output. It MUST NOT be used to identify or match items, and keys need not be unique.

#### Scenario: Key taken from `slug`

- GIVEN an item that is a mapping with `slug: csv-serializer`
- WHEN the key is resolved
- THEN the key is `csv-serializer` and the commit trailer reads `Ralph item 0 (csv-serializer) completed`

#### Scenario: Key falls back to `id` then `name`

- GIVEN an item that is a mapping with `id: 4821` and no `slug`
- WHEN the key is resolved
- THEN the key is `4821`

#### Scenario: Item with no key

- GIVEN an item that is a plain string, or a mapping with no `slug`, `id`, or `name`
- WHEN the key is resolved
- THEN the item has no key and its commit trailer reads `Ralph item 0 completed`
- AND the item is tracked exactly as any keyed item is

#### Scenario: Duplicate keys are not an error

- GIVEN two items that share the same `slug`
- WHEN the project is used
- THEN both items are tracked independently by index and no error is reported

### Requirement: Optional Metadata

A project file MAY carry top-level `slug` and `title` fields, which are read only when the file's top level is a mapping. `slug` names the branch `ralph/<slug>` and falls back to the project file's base name. `title` becomes the pull request title and falls back to the slug.

#### Scenario: Slug and title provided

- GIVEN a project file whose top level is a mapping with `slug: csv-export` and `title: Add CSV export to the reports API`
- WHEN the run starts
- THEN the branch is `ralph/csv-export` and the pull request title is `Add CSV export to the reports API`

#### Scenario: Top-level array has no metadata

- GIVEN a project file at `projects/csv-export.yaml` whose top level is an array
- WHEN the run starts
- THEN the slug is `csv-export`, derived from the file name, and the PR title falls back to that slug

#### Scenario: Metadata omitted from a mapping

- GIVEN a project file whose top level is a mapping with neither `slug` nor `title`
- WHEN the run starts
- THEN both fall back as above and the project is valid

### Requirement: No Completion State In The File

A project file MUST NOT carry completion state. Items MUST NOT have a `passing` field or any equivalent, and nothing — not the author, not ralph, not the AI agent — writes progress back into the file. Completion is recorded in the branch's commit messages; see [get/spec.md](../get/spec.md).

#### Scenario: Items carry no completion field

- GIVEN an item authored for ralph
- WHEN it is written
- THEN it describes work to do and carries no field indicating whether that work is done

#### Scenario: Completion queried from the commit log

- GIVEN a run in progress
- WHEN the author wants to know what is done
- THEN they run `ralph get complete` or `ralph get incomplete`, not read the project file

#### Scenario: Pre-existing completion field is inert

- GIVEN a borrowed project file whose items happen to carry a `passing` field
- WHEN the project is run
- THEN the field is passed through to the agent as ordinary item content and has no effect on completion

### Requirement: Conventional Item Shape

An item authored for ralph SHOULD be a mapping with a `slug` and a `description`, and SHOULD have at least one of `items`, `scenarios`, `code`, or `tests`. This shape is what the skill generates and what the default agent instructions expect; ralph does not enforce it.

- `slug` — lowercase, hyphen-separated label, conventionally unique within the project; becomes the item key
- `description` — what the item covers
- `items` — behavioral outcomes for work that falls outside the spec and orchestration
- `scenarios` — GWT scenarios copied from the spec document
- `code` — code the item should implement, sourced from the orchestration document
- `tests` — tests the item should implement, sourced from the orchestration document

#### Scenario: Item with the conventional shape

- GIVEN an item authored from a spec and orchestration
- WHEN it is written
- THEN it has a `slug`, a `description`, and at least one of `items`, `scenarios`, `code`, or `tests`

#### Scenario: Item with only a slug and description

- GIVEN an item with no `items`, `scenarios`, `code`, or `tests`
- WHEN the project is reviewed
- THEN the item is flagged as giving the agent nothing to build
- AND the run still executes, because ralph enforces no schema

#### Scenario: Behavioral outcomes in `items`

- GIVEN work that falls outside the associated spec and orchestration
- WHEN the author writes the `items` list
- THEN each entry is a specific, observable outcome the agent must achieve, free of architecture decisions such as package names, struct names, or implementation strategies

#### Scenario: Scenarios copied from spec

- GIVEN a project based on a spec document
- WHEN the author writes an item
- THEN relevant scenarios are copied verbatim from the spec into the item's `scenarios` field

### Requirement: Items Are Self-Contained

Each item MUST be written to stand alone. The development agent receives the selected item and the full project file, but not the spec or orchestration documents, so any content the agent needs MUST be present in the item itself.

#### Scenario: Agent context is the item and the project file

- GIVEN an item that references a spec by path only
- WHEN the development agent runs
- THEN the referenced content is unavailable to it, so the item is incomplete as written

#### Scenario: One item is one iteration

- GIVEN work that needs several separate rounds of development
- WHEN the author writes the project
- THEN the work is split across several items, because each iteration works exactly one item

### Requirement: Code and Tests Sourced from Orchestration

An item's `code` and `tests` entries MUST be sourced directly from the feature's orchestration document, never composed freehand. When the feature has no orchestration document, or the orchestration has no matching shape, the field MUST be omitted and `scenarios` and `items` used instead.

Every entry in both fields MUST have `name`, `description`, `module`, and `body`. `module` MUST match a `path` entry in the relevant architecture document.

#### Scenario: Code entry copied from orchestration

- GIVEN an orchestration document defining `ExportReport` in `internal/reports`
- WHEN the author writes the corresponding item
- THEN a `code` entry carries that name, module, description, and body as the orchestration defines them

#### Scenario: No orchestration — code omitted

- GIVEN a feature with no orchestration document
- WHEN the author writes the items
- THEN the `code` and `tests` fields are omitted and the work is expressed with `scenarios` and `items`

#### Scenario: Incomplete code entry

- GIVEN a `code` entry missing `module` or `body`
- WHEN the project is reviewed
- THEN the entry is flagged as incomplete

### Requirement: Helper Items

Each helper function called from a `code` entry's body MUST have its own item with a fully-specified `code` entry. Spec scenarios that directly relate to the helper MUST be copied into that item's `scenarios`, and `items` MUST be used to fill any remaining gaps.

#### Scenario: Helper gets its own item

- GIVEN an orchestration where `ExportReport` calls `buildCSV`
- WHEN the author writes the project
- THEN a separate item exists for `buildCSV` with its own `code` entry carrying `name`, `description`, `module`, and `body`

#### Scenario: Spec scenarios copied to the helper item

- GIVEN a spec scenario that directly describes the behavior of a helper function
- WHEN the author writes the helper item
- THEN that scenario is copied into the helper item's `scenarios` field

#### Scenario: Items fill gaps for the helper

- GIVEN a helper item where the spec and orchestration do not fully describe the expected behavior
- WHEN the author writes the item
- THEN `items` are added to cover the remaining behavioral expectations

### Requirement: Version Bump Items

When the repository uses versioning, the project SHOULD include an item for the version bump. The item MUST specify the bump level — patch, minor, or major — and MUST NOT specify a target version, because ralph determines the current version and applies the bump.

#### Scenario: Bump level specified

- GIVEN a project adding a backwards-compatible feature
- WHEN the author writes the version item
- THEN it reads as a minor bump to the named resource, with no version number

#### Scenario: Independent bumps per resource

- GIVEN a repository with several independently versioned resources
- WHEN the author writes version items
- THEN each resource's bump level is chosen from how its own interface changed

### Requirement: Skill-Format Documentation

The documentation describing the project file format MUST be written as a skill file compatible with Claude Code and opencode. The [Agent Skills | OpenCode](https://opencode.ai/docs/skills/) page defines the format reference.

The skill file MUST begin with YAML frontmatter containing a `name` and `description` field. The body contains the documentation content in markdown.

#### Scenario: Documentation loaded as a skill

- GIVEN an agent that supports Claude Code or opencode skills
- WHEN it loads the project documentation skill
- THEN it can read the full project format and writing rules on demand

#### Scenario: Frontmatter is valid

- GIVEN the documentation skill file
- WHEN the frontmatter is parsed
- THEN `name` is a lowercase hyphen-separated identifier and `description` is a concise one-line summary
