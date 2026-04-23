# Webhook Events Specification

## Purpose

Receive GitHub webhook events for pull requests and dispatch Argo Workflows to implement comment requests or merge approved PRs.

## Requirements

### Requirement: Webhook Endpoint

The service SHALL expose a single `POST /webhook` endpoint for GitHub webhook delivery.

#### Scenario: Valid request

- GIVEN a POST request with a valid JSON payload and correct `X-Hub-Signature-256` header
- WHEN GitHub delivers a supported event
- THEN the event is processed and the server responds with HTTP 200

#### Scenario: Invalid JSON

- GIVEN a POST request with malformed JSON body
- WHEN the webhook is received
- THEN HTTP 400 is returned with an error message

#### Scenario: Unknown repository

- GIVEN a payload from a repository not listed in the server config
- WHEN the webhook is received
- THEN HTTP 401 is returned and no workflow is submitted

### Requirement: Signature Validation

The service MUST validate the `X-Hub-Signature-256` header using HMAC-SHA256 before processing any event.

#### Scenario: Valid signature

- GIVEN the header matches the HMAC-SHA256 of the request body using the configured webhook secret
- WHEN the webhook is received
- THEN the request is processed

#### Scenario: Missing or invalid signature

- GIVEN the `X-Hub-Signature-256` header is absent or does not match
- WHEN the webhook is received
- THEN HTTP 401 is returned and no workflow is submitted

### Requirement: Issue Comment Events

The service SHALL dispatch a Run Workflow for `issue_comment` events on pull requests.

#### Scenario: PR comment accepted

- GIVEN an `issue_comment` event on a pull request from an allowed user
- WHEN the webhook is received
- THEN a Run Workflow is submitted calling `ralph comment` with the comment body, PR number, and branch

#### Scenario: Non-PR issue comment ignored

- GIVEN an `issue_comment` event on a regular issue (not a PR)
- WHEN the webhook is received
- THEN the event is silently ignored and HTTP 200 is returned

### Requirement: Pull Request Review Comment Events

The service SHALL dispatch a Run Workflow for `pull_request_review_comment` events from allowed users.

#### Scenario: Review comment accepted

- GIVEN a `pull_request_review_comment` event from an allowed user
- WHEN the webhook is received
- THEN a Run Workflow is submitted calling `ralph comment` with the comment body

### Requirement: Pull Request Review Events

The service SHALL dispatch a Run Workflow for `commented` reviews and a Merge Workflow for `approved` reviews.

#### Scenario: Review with comment

- GIVEN a `pull_request_review` event with state `commented` and a non-empty body
- WHEN the webhook is received
- THEN a Run Workflow is submitted calling `ralph comment` with the review body

#### Scenario: Review approval triggers merge

- GIVEN a `pull_request_review` event with state `approved`
- WHEN the webhook is received
- THEN a Merge Workflow is submitted calling `ralph merge --local` for the PR branch

#### Scenario: Empty commented review ignored

- GIVEN a `pull_request_review` event with state `commented` and an empty body
- WHEN the webhook is received
- THEN the event is silently ignored and HTTP 200 is returned

### Requirement: User Filtering

The service SHALL filter events based on per-repo allowlists and ignorelists, and a global ralph bot user.

#### Scenario: Ignored user

- GIVEN a comment from a user in `ignoredUsers` or matching `ralphUser`
- WHEN the webhook is received
- THEN the event is silently dropped and HTTP 200 is returned

#### Scenario: Allowlist enforced

- GIVEN `allowedUsers` is non-empty and the author is not in the list
- WHEN the webhook is received
- THEN the event is silently dropped and HTTP 200 is returned

#### Scenario: Open allowlist

- GIVEN `allowedUsers` is empty for the repo
- WHEN the webhook is received from a non-ignored user
- THEN the event is accepted regardless of author

### Requirement: Workflow Submission

The service SHALL submit Argo Workflows asynchronously so webhook responses are not delayed.

#### Scenario: Async submission

- GIVEN a valid, accepted event
- WHEN the event is dispatched
- THEN HTTP 200 is returned immediately
- AND the Argo Workflow is submitted in a background goroutine

### Requirement: Project File Derivation

The service SHALL derive the project file path from the PR branch name.

#### Scenario: Ralph branch convention

- GIVEN a PR with head branch `ralph/my-feature`
- WHEN an event is dispatched
- THEN the project file is resolved to `projects/my-feature.yaml`

#### Scenario: Non-ralph branch

- GIVEN a PR with head branch `feat/some-work`
- WHEN an event is dispatched
- THEN the project file is resolved to `projects/feat-some-work.yaml` (slashes replaced with dashes)
