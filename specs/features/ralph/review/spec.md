# Review Specification

## Purpose

Run AI-powered code reviews against standards defined in `.ralph/config.yaml`, producing project files and pull requests with findings.

## Requirements

### Requirement: Review Execution

The system SHALL run each review item as an AI agent prompt and commit any resulting changes.

#### Scenario: Remote submission (default)

- GIVEN review items are configured and the current branch is in sync with the remote
- WHEN the user runs `ralph review`
- THEN a review Argo Workflow is submitted to the Kubernetes cluster

#### Scenario: Local execution

- GIVEN the `--local` flag is set
- WHEN the user runs `ralph review --local`
- THEN each review item runs as an AI agent on the local machine in shuffled order
- AND any uncommitted changes after each item are committed to a branch derived from the modified project file

#### Scenario: No review items configured

- GIVEN `review.items` is empty in `.ralph/config.yaml`
- WHEN the user runs `ralph review`
- THEN an error is returned before any agent runs

### Requirement: Review Item Sources

Each review item MUST specify exactly one content source: `text`, `file`, or `url`.

#### Scenario: Inline text

- GIVEN a review item with `text: "All functions must have tests."`
- WHEN the item is processed
- THEN the text is used directly as the review prompt content

#### Scenario: File source

- GIVEN a review item with `file: docs/standards.md`
- WHEN the item is processed
- THEN the file is read from the repo root and its contents are used as the prompt

#### Scenario: URL source

- GIVEN a review item with `url: https://example.com/guide`
- WHEN the item is processed
- THEN the URL is fetched and the response body is used as the prompt

### Requirement: Loop Items

The system SHALL support loop review items that expand over entries from `architecture.yaml`.

#### Scenario: Loop expansion

- GIVEN a review item with `loop: <query>` referencing function paths in `architecture.yaml`
- WHEN the item is processed
- THEN one AI agent iteration runs per matching function in the architecture file
- AND each iteration receives the function name and path as context

### Requirement: Item Filtering

The system SHOULD allow running a subset of review items via `--filter` or `--one`.

#### Scenario: Filter by keyword

- GIVEN `--filter testing` is set
- WHEN review runs
- THEN only items whose `text`, `file`, `url`, or `loop` fields contain "testing" (case-insensitive) are executed

#### Scenario: No matches for filter

- GIVEN `--filter` is set and no items match
- WHEN review runs
- THEN an error is returned before any agent runs

#### Scenario: One random item

- GIVEN `--one` is set
- WHEN review runs
- THEN exactly one item is selected at random from the shuffled list and executed

### Requirement: Randomized Order

The system SHALL process review items in a randomized order each run.

#### Scenario: Reproducible shuffle

- GIVEN `--seed <N>` is set
- WHEN review runs
- THEN items are shuffled using `N` as the random seed, producing the same order each time

#### Scenario: Random seed

- GIVEN `--seed` is not set
- WHEN review runs
- THEN a random seed is chosen and logged so the run can be reproduced

### Requirement: Pull Request Creation

The system SHOULD create a pull request with an AI-generated summary after local review completes.

#### Scenario: PR created

- GIVEN the review produced commits ahead of the base branch
- WHEN local execution finishes
- THEN a pull request is opened with an AI-generated body summarizing findings

#### Scenario: No changes

- GIVEN no project file was modified and no commits were produced
- WHEN local execution finishes
- THEN no PR is created

### Requirement: Architecture Generation

The system SHALL generate an `architecture.yaml` file via `ralph review architecture`.

#### Scenario: Successful generation

- GIVEN the user runs `ralph review architecture`
- WHEN the AI agent finishes
- THEN `architecture.yaml` is written at the repo root
- AND the file is validated; up to 3 fix attempts are made if validation fails

#### Scenario: PR in workflow context

- GIVEN ralph is running inside an Argo Workflow container
- WHEN architecture generation completes and changes exist
- THEN the file is committed to an `architecture` branch and a PR is opened

### Requirement: Model Selection

The system SHOULD use the review-specific model when configured, falling back to the global model.

#### Scenario: Review model override

- GIVEN `review.model` is set in `.ralph/config.yaml`
- WHEN review runs
- THEN the review model is used unless `--model` overrides it at the command line
