# Comment Specification

## Purpose

Handle a single comment-triggered development iteration, running the AI agent against instructions derived from a PR comment body.

## Requirements

### Requirement: Comment Execution

The system SHALL run one AI agent iteration in response to a PR comment body.

#### Scenario: Successful iteration

- GIVEN a valid project file exists for the given branch and the comment body is non-empty
- WHEN `ralph comment <body> --repo <owner/repo> --branch <branch> --pr <number>` is run
- THEN the AI agent is invoked with a rendered prompt incorporating the comment body, PR number, branch, repo owner, and repo name
- AND the agent runs to completion

#### Scenario: Project file not found

- GIVEN the branch is `ralph/my-feature` but `projects/my-feature.yaml` does not exist
- WHEN `ralph comment` is run
- THEN an error is returned before the agent is invoked

#### Scenario: Invalid project file

- GIVEN the project file exists but fails to parse
- WHEN `ralph comment` is run
- THEN an error is returned before the agent is invoked

### Requirement: Instruction Rendering

The system SHALL render the AI prompt from the configured comment instructions template, substituting PR context variables.

#### Scenario: Template variables injected

- GIVEN a comment instructions template containing `{{.CommentBody}}`, `{{.PRNumber}}`, `{{.PRBranch}}`, `{{.RepoOwner}}`, or `{{.RepoName}}`
- WHEN the prompt is rendered
- THEN each placeholder is replaced with the corresponding value from the command arguments

#### Scenario: Custom instructions

- GIVEN `commentInstructionsFile` is configured in the webhook app config and its content has been passed to the container
- WHEN the prompt is rendered
- THEN the custom template is used instead of the built-in default

#### Scenario: Template parse failure

- GIVEN the instructions template contains invalid Go template syntax
- WHEN the prompt is rendered
- THEN the raw template text is used as the prompt without substitution

### Requirement: Service Management

The system SHOULD start configured services before the agent runs and stop them after, unless `--no-services` is set.

#### Scenario: Services started

- GIVEN `services` are defined in `.ralph/config.yaml` and `--no-services` is not set
- WHEN `ralph comment` runs
- THEN services are started before the AI agent and stopped when the command exits

#### Scenario: No-services flag

- GIVEN `--no-services` is set
- WHEN `ralph comment` runs
- THEN no services are started or stopped regardless of config

### Requirement: No Branch or PR Side Effects

The system SHALL NOT create branches, commits, or pull requests as part of comment execution.

#### Scenario: Changes left uncommitted

- GIVEN the AI agent makes file changes
- WHEN the agent finishes
- THEN no automatic commit or push is performed by `ralph comment`
- AND no pull request is created or updated
