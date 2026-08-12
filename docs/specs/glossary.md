# Glossary

Terms used throughout these documents. A runner that executes projects will have its own vocabulary for the mechanics of doing so — how an item array is resolved, how completion is recorded — and that belongs in the runner's documentation, not here.

## Component

A top-level deployment or ownership boundary — a distinct service, app, or library that could be developed and deployed independently. Good component names reflect runtime identity (`api`, `worker`, `frontend`), not internal organization.

## Feature

A coherent slice of user-facing or system-facing behavior — something a user can do, or something the system does on their behalf. Good feature names describe what the system does (`auth`, `payments`, `notifications`), not how it does it (`jwt-handler`, `stripe-client`). If a feature grows too large to read comfortably, split it by sub-feature rather than by implementation detail.

## Requirement

A single behavior the system must have, stated as observable behavior rather than implementation. Requirements live in a [spec](spec.md) and use RFC 2119 keywords (SHALL, MUST, SHOULD, MAY) to signal strength.

## Scenario

A concrete, testable example of a requirement in action, written in Given/When/Then form. Scenarios are copied verbatim from a spec into a [project](project.md) so the implementing agent sees them without reading the spec.

## Project

A YAML or JSON file containing an array of work items, drafted from a spec, an architecture, and an orchestration. See [Project Format](project.md).

## Item

One element of a project's array, and the unit of one iteration. An item must be self-contained: the agent implementing it sees the item and the project file, not the source documents it was drawn from.

## Deep Module

A module with a simple interface but complex implementation. Deep modules hide implementation complexity behind a clean, minimal API, providing powerful functionality without exposing internal details. This design principle maximizes the benefit-to-complexity ratio by minimizing the cognitive load on users while maximizing utility.

## Pure Module

A module that contains only value objects and pure functions — code with no side effects. Pure modules do not perform I/O, mutate shared state, or call external services. They are fully testable with unit tests alone, with no need for mocks or integration setup.

## Implementation Module

A module that contains concrete technical implementation details and low-level operations. Implementation modules execute specific tasks such as database queries, API calls, cryptographic operations, file I/O, or data transformations. These modules provide the actual "how" of executing operations rather than coordinating what operations to execute.

Each implementation module covers a single deep concern — one cohesive area of functionality with a simple interface over hidden complexity.

## Orchestration Module

A module that contains only domain logic for coordinating other modules. Orchestration modules define workflows, manage execution sequences, enforce business rules, and delegate to implementation modules. They describe "what" should happen and "when" without containing the low-level details of "how" operations are performed.

A small app typically contains a single orchestration module. As it grows, the orchestration module should be split along deep concern boundaries — each resulting module coordinates one deep concern.

## Category

The classification a module is assigned in [`specs/architecture.yaml`](architecture.md), declaring what kinds of code resource belong in it. `entry`, `orchestration`, `implementation`, and `pure` are the common baseline; projects add their own where they have a distinct kind of module.
