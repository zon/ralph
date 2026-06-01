# Architecture Format

The architecture format is used to outline the [deep modules](../glossary.md#deep-module) of an application in YAML.

## Location

Architecture documents live in two places:

- **`/specs/architecture.yaml`** — describes the **current** modules of the application as a whole.
- **`/specs/features/<component>/<feature>/architecture.yaml`** (optional) — describes **future** modules introduced or changed by a specific feature. Omit when the feature does not introduce new modules.

## Format

Architecture documents use YAML format with the following structure:

### Top-level fields

- **categories** (required, list): Declares the module categories used throughout this architecture document. Each category object has the following fields:

  - **slug** (required, string): Short identifier used as the value of `category` on each module. Must be unique within the document.
  - **description** (required, string): One sentence describing what this category of module is for.
  - **orchestration** (optional, boolean): Whether modules in this category are [orchestration modules](../glossary.md#orchestration-module) rather than [implementation modules](../glossary.md#implementation-module). Defaults to `false` if omitted.
  - **signatures** (required, list of strings): The types of code resources that should be found in modules of this category (e.g. exported functions, interfaces, struct types, CLI wrappers).

- **modules** (required, list): The modules that make up the architecture. Each module has the following fields:

  - **path** (required, string): The file path or directory path where the module is located or should be implemented. Relative to the repo root.
  - **description** (required, string): A single short sentence stating the module's purpose and role. Do not include method names, route lists, interface names, or error types — details like these churn every time the module grows. A good description should survive multiple features being added without needing an edit.
  - **category** (required, string): The slug of the category this module belongs to.

## Example

```yaml
categories:
  - slug: entry
    description: Main package that wires real dependencies and starts the application.
    signatures:
      - main function
      - dependency wiring

  - slug: orchestration
    description: Domain logic modules that define and coordinate core business processes.
    orchestration: true
    signatures:
      - domain logic
      - domain logic integration tests

  - slug: implementation
    description: Real dependency implementations and mocks that back the domain interfaces.
    signatures:
      - real dependency implementations
      - mocks
      - unit tests

modules:
  - path: cmd/myapp
    description: Wires real dependencies into the application and starts the server.
    category: entry

  - path: internal/orders
    description: Orchestrates order placement, fulfillment, and cancellation workflows.
    category: orchestration

  - path: internal/inventory
    description: Manages stock levels, reservations, and availability checks.
    category: orchestration

  - path: internal/postgres
    description: PostgreSQL-backed implementations of the domain repository interfaces.
    category: implementation

  - path: internal/httpapi
    description: HTTP handlers that translate requests into domain calls and format responses.
    category: implementation
```
