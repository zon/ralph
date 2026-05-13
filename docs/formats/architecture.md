# Architecture Format

The architecture format is used to outline the [deep modules](../glossary.md#deep-module) of an application in YAML.

## Location

Architecture documents live in two places:

- **`/specs/architecture.yaml`** — describes the **current** modules of the application as a whole.
- **`/specs/features/<component>/<feature>/architecture.yaml`** (optional) — describes **future** modules introduced or changed by a specific feature. Omit when the feature does not introduce new modules.

## Format

Architecture documents use YAML format with the following structure:

### Fields

- **path** (required, string): The file path or directory path where the module is located or should be implemented. This should be relative to the repo root.

- **description** (required, string): A clear explanation of what the module does, its responsibilities, and how it fits into the overall architecture. Should be detailed enough to understand the module's purpose and scope.

- **orchestration** (optional, boolean): When set to `true`, indicates this is an [orchestration module](../glossary.md#orchestration-module) rather than an [implementation module](../glossary.md#implementation-module). Defaults to `false` if omitted.

## Example

```yaml
modules:
  - path: src/services
    description: Contains business logic for orders, inventory, payments, and customer management. Defines multi-step processes like order fulfillment and user onboarding, enforces domain rules, and manages transaction boundaries.
    orchestration: true

  - path: src/api
    description: API layer that handles HTTP requests and responses. Contains route definitions, request validation, response formatting, and maps HTTP operations to service calls.

  - path: src/repositories
    description: Data access layer implementing repository patterns for entity persistence. Executes database queries, handles object-relational mapping, and manages data retrieval and storage logic.

  - path: src/auth
    description: Authentication and authorization module. Handles user identity verification, token generation and validation, session management, and permission checking with cryptographic operations.

  - path: src/queue
    description: Message queue integration for asynchronous task processing. Manages job scheduling, message publishing/consuming, serialization, and delivery guarantees.

  - path: src/notifications
    description: Notification delivery system supporting multiple channels (email, SMS, push). Handles templating, formatting, provider integration, and delivery tracking.

  - path: src/utils
    description: Shared utility functions and helpers used across the application. Includes string manipulation, date handling, validation, and common algorithms.
```
