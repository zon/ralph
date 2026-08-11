# Glossary

## Project

Any YAML or JSON file that contains an array of work items. Ralph imposes no schema on a project file — it reads an array out of it with the item query and treats everything else as opaque context for the AI agent. See [Project Format](formats/project.md).

## Item

One element of a project's resolved array, and the unit of one iteration. An item can be any value — a string, a mapping, a nested structure. Ralph identifies it by its index in the array, which is stable because the project file is not written to during a run.

## Item Query

A [jq](https://jqlang.org/manual/) expression that resolves a project file to its item array. Defaults to `.`, which fits a file whose top level is already an array. Configured with `items` in `.ralph/config.yaml` or `--items`.

## Item Key

The value of an item's `slug`, `id`, or `name` field, checked in that order. A convenience label that names the item in commit messages, logs, and picker output. It is not an identifier — items are tracked by index whether or not they have a key.

## Completion Trailer

The last line of an iteration's commit message, written by the development agent to record that an item is finished — `Ralph item <index> completed`, or `Ralph item <index> (<key>) completed` when the item has a key. The set of trailers on a branch is ralph's only record of progress. See [Iterations](iterations.md).

## Component

A top-level deployment or ownership boundary — a distinct service, app, or library that could be developed and deployed independently. Good component names reflect runtime identity (`api`, `worker`, `frontend`), not internal organization.

## Deep Module

A module with a simple interface but complex implementation. Deep modules hide implementation complexity behind a clean, minimal API, providing powerful functionality without exposing internal details. This design principle maximizes the benefit-to-complexity ratio by minimizing the cognitive load on users while maximizing utility.

## Feature

A coherent slice of user-facing or system-facing behavior — something a user can do, or something the system does on their behalf. Good feature names describe what the system does (`auth`, `payments`, `notifications`), not how it does it (`jwt-handler`, `stripe-client`). If a feature grows too large to read comfortably, split it by sub-feature rather than by implementation detail.

## Implementation Module

A module that contains concrete technical implementation details and low-level operations. Implementation modules execute specific tasks such as database queries, API calls, cryptographic operations, file I/O, or data transformations. These modules provide the actual "how" of executing operations rather than coordinating what operations to execute.

Each implementation module covers a single deep concern — one cohesive area of functionality with a simple interface over hidden complexity.

## Pure Module

A module that contains only value objects and pure functions — code with no side effects. Pure modules do not perform I/O, mutate shared state, or call external services. They are fully testable with unit tests alone, with no need for mocks or integration setup.

## Orchestration Module

A module that contains only domain logic for coordinating other modules. Orchestration modules define workflows, manage execution sequences, enforce business rules, and delegate to implementation modules. They describe "what" should happen and "when" without containing the low-level details of "how" operations are performed.

A small app typically contains a single orchestration module. As it grows, the orchestration module should be split along deep concern boundaries — each resulting module coordinates one deep concern.
