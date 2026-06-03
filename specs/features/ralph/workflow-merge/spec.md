# Workflow Merge Specification

## Purpose

`ralph workflow merge` runs pre-merge operations and performs the merge after the workspace is ready.

## Requirements

### Requirement: Workspace Setup

The system SHALL prepare the container workspace as defined in [workflow-workspace/spec.md](../workflow-workspace/spec.md) before doing any work, with the PR branch as the checkout target and symlink setup disabled.

### Requirement: Completed Project Cleanup

Before merging, the system SHALL delete any project files where all requirements are passing, commit the deletion, and push to the remote.

#### Scenario: Completed project files deleted

- GIVEN one or more project files in `projects/` have all requirements `passing: true`
- WHEN `ralph workflow merge` runs
- THEN the completed project files are deleted, the deletion is committed, and the commit is pushed before merging

#### Scenario: No completed projects

- GIVEN no project files are fully passing
- WHEN `ralph workflow merge` runs
- THEN no files are deleted and execution proceeds directly to the merge

### Requirement: GitHub Head Synchronization

Before merging, the system SHALL confirm that GitHub has processed the pushed commit. If GitHub does not reflect the push within a reasonable timeout, the merge SHALL be aborted.

#### Scenario: GitHub reflects updated SHA

- GIVEN a commit was pushed to the PR branch
- WHEN the merge step verifies GitHub has processed the push
- THEN the merge proceeds once GitHub reports the expected SHA

#### Scenario: Sync timeout

- GIVEN GitHub does not reflect the pushed commit within the allowed timeout
- WHEN the merge step verifies GitHub has processed the push
- THEN an error is returned and the PR merge is not attempted

### Requirement: PR Merge

The system SHALL merge the PR into the base branch after cleanup and synchronization are complete.

#### Scenario: PR merged

- GIVEN the workspace is ready and any cleanup has been committed and pushed
- WHEN `ralph workflow merge` runs
- THEN the PR is merged into the base branch

