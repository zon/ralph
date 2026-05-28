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

- **description** (required, string): A single short sentence stating the module's purpose and role. Do not include method names, route lists, interface names, or error types — details like these churn every time the module grows. A good description should survive multiple features being added without needing an edit.

- **orchestration** (optional, boolean): When set to `true`, indicates this is an [orchestration module](../glossary.md#orchestration-module) rather than an [implementation module](../glossary.md#implementation-module). Defaults to `false` if omitted.

## Example

```yaml
modules:
  - path: src/services
    description: Orchestrates multi-step business processes for orders, inventory, payments, and customer management.
    orchestration: true

  - path: src/api
    description: HTTP layer that maps incoming requests to service calls and formats responses.

  - path: src/repositories
    description: Data access layer that persists and retrieves domain entities.

  - path: src/auth
    description: Handles user identity verification, token lifecycle, and permission checking.

  - path: src/queue
    description: Message queue integration for asynchronous task scheduling and delivery.

  - path: src/notifications
    description: Delivers notifications across email, SMS, and push channels.

  - path: src/utils
    description: Shared utility functions used across the application.
```
