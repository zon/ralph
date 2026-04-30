# Workflow Specification

## Purpose

Execute ralph inside an Argo Workflow container, handling repository setup, GitHub authentication, base-branch synchronization, and the full project or review run.

## Requirements

### Requirement: Container Bootstrap

The system SHALL authenticate to GitHub, configure git, and clone the repository before doing any work.

#### Scenario: GitHub App token setup

- GIVEN GitHub App credentials mounted at `/secrets/github`
- WHEN the workflow container starts
- THEN a GitHub App installation token is generated and git HTTPS authentication is configured for the target repo

#### Scenario: OpenCode credential setup

- GIVEN OpenCode provider credentials are available in the workspace
- WHEN the workflow container starts
- THEN the credentials are placed in the expected location for the AI agent to use

#### Scenario: Git user configuration

- GIVEN `--bot-name` and `--bot-email` are provided (defaults: `ralph-zon[bot]` and `ralph-zon[bot]@users.noreply.github.com`)
- WHEN the workflow container starts
- THEN git is configured with those identity values for all subsequent commits

#### Scenario: Repository cloning

- GIVEN `GIT_BRANCH` environment variable is set to the branch to clone
- WHEN the workspace is prepared
- THEN the repository is cloned at that branch into `/workspace`

### Requirement: Workspace Symlink Setup

The system SHALL symlink mounted ConfigMap and Secret files into the working directory after cloning.

#### Scenario: Linked ConfigMap

- GIVEN a ConfigMap mount with `link: true` in `.ralph/config.yaml`
- WHEN setup-workspace runs inside the container
- THEN a symlink is created from the repo working directory path to the mounted path under `/workspace`

#### Scenario: Already-linked path

- GIVEN a symlink target already exists at the destination
- WHEN setup-workspace runs
- THEN the existing symlink is left untouched and no error is raised

### Requirement: Base Branch Synchronization

The system SHALL attempt to merge the base branch into the project branch before running the project.

#### Scenario: Branch is up-to-date

- GIVEN the project branch's merge-base with the base branch equals the base branch tip
- WHEN synchronization runs
- THEN no merge is performed and execution continues

#### Scenario: Clean merge

- GIVEN the project branch is behind the base branch with no conflicts
- WHEN synchronization runs
- THEN the base branch is merged into the project branch (fast-forward or auto-merge)
- AND execution continues

#### Scenario: Merge conflicts resolved by AI

- GIVEN a merge attempt produces conflicts
- WHEN the merge fails
- THEN the merge is aborted
- AND an AI agent is invoked with instructions to resolve all conflicts, run tests, and stage the resolved files
- AND execution continues after resolution

#### Scenario: Base branch fetch failure

- GIVEN the base branch cannot be fetched (e.g., network error)
- WHEN synchronization runs
- THEN a warning is logged and execution continues without merging

### Requirement: Project Execution Mode

The system SHALL run the full project iteration loop when invoked without `--review`. Each iteration follows the behavior defined in [execution.md](execution.md).

#### Scenario: Project run

- GIVEN `--project-branch` and a project path are provided
- WHEN the workflow container runs
- THEN the project file is loaded, the iteration loop executes locally, and a PR is created if changes were made

#### Scenario: Missing project path

- GIVEN no project path argument and `--review` is not set
- WHEN the workflow container runs
- THEN an error is returned before any work is done

### Requirement: Review Execution Mode

The system SHALL run the review loop when `--review` is set, skipping project execution. See [review.md](review.md) for review item behavior.

#### Scenario: Review run

- GIVEN `--review` is set
- WHEN the workflow container runs
- THEN the review loop runs with `--local`, base branch synchronization is skipped, and any resulting PR is created

#### Scenario: Review with filter

- GIVEN `--review` and `--filter <keyword>` are set
- WHEN the workflow container runs
- THEN only review items matching the keyword are processed

### Requirement: Mutex-Based Concurrency Control

The system SHALL use an Argo Workflow mutex keyed on the project branch name to prevent concurrent runs on the same branch.

#### Scenario: Concurrent submission

- GIVEN a workflow is already running for branch `ralph/my-feature`
- WHEN a second workflow is submitted for the same branch
- THEN the second workflow waits until the first completes before executing

### Requirement: Workflow Lifecycle

Workflows SHALL be automatically cleaned up after completion.

#### Scenario: TTL expiry

- GIVEN a workflow has completed (success or failure)
- WHEN 24 hours have elapsed since completion
- THEN the workflow resource is deleted from Kubernetes

#### Scenario: Pod cleanup

- GIVEN a workflow has completed
- WHEN 10 minutes have elapsed since completion
- THEN the workflow pods are garbage-collected

### Requirement: Merge Workflow

The system SHALL support a separate merge workflow that cleans up completed project files and merges the PR.

#### Scenario: Merge with completed projects

- GIVEN one or more project files in `projects/` have all requirements `passing: true`
- WHEN `ralph merge --local` runs inside the container
- THEN the complete project files are deleted, the deletion is committed and pushed, and GitHub is polled until its view of the head SHA matches before merging the PR

#### Scenario: No completed projects

- GIVEN no project files are fully passing
- WHEN `ralph merge --local` runs
- THEN no files are deleted and the PR is merged directly

#### Scenario: GitHub head sync timeout

- GIVEN a push was made but GitHub does not reflect the updated SHA within 60 seconds (20 attempts × 3 s)
- WHEN the merge step polls for sync
- THEN an error is returned and the PR merge is not attempted

### Requirement: Debug Mode

The system SHOULD support a debug mode that clones a specific ralph branch and invokes ralph via `go run` instead of the built binary.

#### Scenario: Debug branch set

- GIVEN `--debug <branch>` is provided
- WHEN the workflow container starts
- THEN the specified ralph source branch is checked out into `/workspace/ralph`
- AND ralph is invoked via `go run ./cmd/ralph` instead of the installed binary

### Requirement: AI Token Statistics

The system SHOULD display OpenCode token usage statistics after each workflow execution.

#### Scenario: Stats display

- GIVEN any workflow run completes (project or review)
- WHEN execution finishes
- THEN accumulated AI token usage statistics are printed to the log
