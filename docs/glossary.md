# Glossary

## Component

A top-level deployment or ownership boundary — a distinct service, app, or library that could be developed and deployed independently. Good component names reflect runtime identity (`api`, `worker`, `frontend`), not internal organization.

## Deep Module

A module with a simple interface but complex implementation. Deep modules hide implementation complexity behind a clean, minimal API, providing powerful functionality without exposing internal details. This design principle maximizes the benefit-to-complexity ratio by minimizing the cognitive load on users while maximizing utility.

## Feature

A coherent slice of user-facing or system-facing behavior — something a user can do, or something the system does on their behalf. Good feature names describe what the system does (`auth`, `payments`, `notifications`), not how it does it (`jwt-handler`, `stripe-client`). If a feature grows too large to read comfortably, split it by sub-feature rather than by implementation detail.

## Implementation Module

A module that contains concrete technical implementation details and low-level operations. Implementation modules execute specific tasks such as database queries, API calls, cryptographic operations, file I/O, or data transformations. These modules provide the actual "how" of executing operations rather than coordinating what operations to execute.

Each implementation module covers a single deep concern — one cohesive area of functionality with a simple interface over hidden complexity.

## Orchestration Module

A module that contains only domain logic for coordinating other modules. Orchestration modules define workflows, manage execution sequences, enforce business rules, and delegate to implementation modules. They describe "what" should happen and "when" without containing the low-level details of "how" operations are performed.

A small app typically contains a single orchestration module. As it grows, the orchestration module should be split along deep concern boundaries — each resulting module coordinates one deep concern.
