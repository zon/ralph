# Validate Command Specification

## Purpose

Define the behavior of `ralph validate`, which checks that a project file is usable by a run: that it parses as YAML or JSON, that the item query evaluates against it, and that the query resolves to at least one non-empty item. When the file fails to parse, the command asks a locally-run agent to repair it and retries. On success the file is rewritten in canonical YAML. A `.json` input is written to a new `.yaml` file and the original is removed.

Ralph imposes no schema on a project file, so validation makes no schema checks. Anything that parses and yields non-empty items is a valid project.

## Requirements

### Requirement: Command Invocation

The system SHALL provide a `ralph validate <file>` subcommand that accepts a path to a project file (JSON or YAML) as its sole positional argument.

#### Scenario: File path provided

- GIVEN a path to a project file
- WHEN the user runs `ralph validate <file>`
- THEN the command loads the file from that path and begins validation

#### Scenario: File path missing

- GIVEN no positional argument
- WHEN the user runs `ralph validate`
- THEN the command exits with a non-zero status and reports that a project file path is required

### Requirement: Item Query Resolution

The command SHALL resolve the item query with the same two-level precedence a run uses: `--items` at the command line takes priority. Otherwise the `items` field in `.ralph/config.yaml` is used. Otherwise the query defaults to `.`. Validating with one query and running with another does not check what the run will do.

#### Scenario: `--items` flag takes precedence

- GIVEN `items: .requirements` is set in `.ralph/config.yaml`
- AND the user runs `ralph validate <file> --items '.spec.tasks'`
- WHEN the item query is resolved
- THEN `.spec.tasks` is evaluated against the parsed file

#### Scenario: Config query used when no flag is passed

- GIVEN `items: .requirements` is set in `.ralph/config.yaml`
- AND no `--items` flag is passed
- WHEN the item query is resolved
- THEN `.requirements` is evaluated against the parsed file

#### Scenario: Default query when flag and config are unset

- GIVEN `items` is not set in `.ralph/config.yaml`
- AND no `--items` flag is passed
- WHEN the item query is resolved
- THEN the query `.` is evaluated, so a file whose top level is an array validates with no configuration

### Requirement: Validation Checks

The command MUST check exactly three things, in order: the file parses as YAML or JSON, the item query evaluates against the parsed document without error, and the query resolves to at least one non-empty item. There MUST be no schema check: no field is required, and no field is rejected.

#### Scenario: Well-formed project

- GIVEN a file that parses and whose query resolves to one or more items
- WHEN `ralph validate <file>` is run
- THEN all three checks pass on the first attempt
- AND no agent is invoked

#### Scenario: Unrecognized fields accepted

- GIVEN a file that parses and yields items but contains fields Ralph does not read
- WHEN `ralph validate <file>` is run
- THEN validation succeeds and the unrecognized fields are left untouched

#### Scenario: Items with no conventional shape accepted

- GIVEN a file whose top level is an array of plain strings
- WHEN `ralph validate <file>` is run with the default query
- THEN validation succeeds, because each string is a valid item

#### Scenario: Parse failure

- GIVEN a file that is not valid YAML or JSON
- WHEN `ralph validate <file>` is run
- THEN the command enters the fix loop described below
- AND the underlying parse error is reported to the user before each fix attempt

#### Scenario: Query evaluation failure

- GIVEN a file that parses and a query that cannot be evaluated against it
- WHEN `ralph validate <file>` is run
- THEN the command exits with a non-zero status reporting the query error
- AND no agent is invoked, because the file is not the thing that is wrong

#### Scenario: Query yields no items

- GIVEN a file that parses and a query that produces no output
- WHEN `ralph validate <file>` is run
- THEN the command exits with a non-zero status and the error names the query: `item query yielded no items: <query>`
- AND no agent is invoked

#### Scenario: Query yields only empty items

- GIVEN a file that parses and whose item list holds nothing but nulls, blank strings, and empty mappings
- WHEN `ralph validate <file>` is run
- THEN the resolved list is empty, so the command exits with a non-zero status and the error names the query: `item query yielded no items: <query>`
- AND no agent is invoked, because the file parses

#### Scenario: Empty entries among real items pass validation

- GIVEN a file that parses and whose item list holds two work items with a null entry between them
- WHEN `ralph validate <file>` is run
- THEN validation passes, because resolution drops the null and two non-empty items remain

### Requirement: Local Agent Fix Loop

When the file fails to parse, the command MUST invoke an AI agent locally to repair the file in place, then retry parsing. The loop MUST continue until the file parses or the attempt limit is reached.

The agent MUST be run locally on the current machine (the `local` execution mode, selected with `--mode local`), never delegated to a remote workflow runner. The agent MUST use the model resolved from the Ralph config file, with no command-line override required.

The shared `--model` and `--variant` options contract is defined in [model-options.md](model-options.md). `ralph validate` does not expose those options and resolves its model from config alone.

Model resolution follows a two-level precedence: if `validate.model` is set in `.ralph/config.yaml` that model is used. Otherwise the top-level `model` field is used as the fallback.

The repair prompt SHALL run with opencode's primary agent and SHALL NOT receive the configured agent, so a configured agent that denies file writes cannot block the repair. This scoping leaves the model resolution above unchanged.

#### Scenario: Agent fixes a malformed file

- GIVEN a file whose contents do not parse as YAML or JSON
- WHEN `ralph validate <file>` is run
- THEN the command invokes the agent with the file path and the parse error
- AND the agent rewrites the file
- AND the command retries parsing against the updated file

#### Scenario: Local execution

- GIVEN any failed parse
- WHEN the fix loop invokes the agent
- THEN the agent runs on the local machine using the same path used by the `local` execution mode of `ralph run`
- AND no Argo workflow or remote runner is involved

#### Scenario: Validate-specific model used when configured

- GIVEN `validate.model` is set in `.ralph/config.yaml`
- WHEN the fix loop invokes the agent
- THEN the validate-specific model is used
- AND the user is not required to pass a model flag

#### Scenario: Fallback to main model when validate model is unset

- GIVEN `validate.model` is not set in `.ralph/config.yaml`
- AND the top-level `model` field is set
- WHEN the fix loop invokes the agent
- THEN the top-level model is used as the fallback

#### Scenario: Repair prompt runs without the configured agent

- GIVEN the agent resolves to `build`
- WHEN the fix loop invokes the repair prompt
- THEN the `--agent` option is omitted from the opencode invocation
- AND opencode's primary agent is used
- AND the validate-specific or fallback model is still used

#### Scenario: Query checks run after the file parses

- GIVEN a file that the agent repairs into valid YAML
- WHEN the fix loop exits
- THEN the item query is evaluated against the repaired file
- AND a query that yields no non-empty items fails validation without re-entering the fix loop

### Requirement: Fix Loop Limit

The fix loop MUST be capped at 10 total parse attempts (the initial attempt plus up to 9 agent-assisted retries). If the file still fails to parse after the final attempt, the command MUST exit with a non-zero status code and report that the limit was reached along with the most recent parse error.

#### Scenario: File becomes parseable within the limit

- GIVEN a file that the agent successfully repairs within 10 attempts
- WHEN `ralph validate <file>` is run
- THEN the loop exits as soon as parsing succeeds
- AND the command proceeds to the query checks

#### Scenario: Limit exceeded

- GIVEN a file that the agent cannot repair in 10 attempts
- WHEN `ralph validate <file>` is run
- THEN the command exits with a non-zero status
- AND the error message reports that the 10-attempt limit was reached
- AND the error message includes the final parse error

### Requirement: Canonical Formatting

After all three checks pass, the command MUST write the parsed document back to disk as YAML. Because Ralph has no project schema, the rewrite MUST preserve the document's full content and key order. Only formatting is normalized. No field is dropped for being unrecognized, and no field is added.

#### Scenario: File rewritten in canonical format

- GIVEN a project file that validates (immediately or after agent fixes)
- WHEN `ralph validate <file>` finishes validation
- THEN the file is rewritten as canonical YAML
- AND the on-disk content parses to the same document as the input

#### Scenario: Already-canonical YAML file is unchanged

- GIVEN a YAML file that is already in canonical format
- WHEN `ralph validate <file>` rewrites it
- THEN the resulting file content is byte-identical to the input

#### Scenario: Unrecognized fields survive the rewrite

- GIVEN a project file containing fields Ralph does not read
- WHEN the file is rewritten in canonical format
- THEN those fields are present in the output with their original values

#### Scenario: JSON file renamed to YAML

- GIVEN a project file with a `.json` extension that validates
- WHEN `ralph validate <file>` finishes validation
- THEN the document is written to a new file with the same name but a `.yaml` extension
- AND the original `.json` file is removed

### Requirement: Successful Validation Output

When validation completes successfully, the command MUST exit with status code 0 and emit a confirmation message identifying the file path and the number of items the query resolved.

#### Scenario: Valid project file

- GIVEN a project file that ends up valid (with or without agent fixes)
- WHEN `ralph validate <file>` finishes
- THEN the command exits with status code 0
- AND a message confirms the project is valid and reports the file path and its item count

### Requirement: Validation Is Not Required to Run

Validation MUST be an optional convenience, not a precondition for a run. A run resolves items with the same query and reports the same errors, so a project file that has never been validated is runnable.

#### Scenario: Foreign project file run without validation

- GIVEN a project file owned by another tool, whose formatting must not be rewritten
- WHEN the user runs Ralph against it without validating first
- THEN the run resolves items normally and the file is not reformatted
