# Architecture Format

The architecture format lists, in YAML, the components a project is built from.

## Location

The architecture document lives at **`/specs/architecture.yaml`** and describes the **current** components of the application.

## Format

The architecture document is a YAML list. A component is a collection of resources that serve one concern: source code, scripts, runtimes, assets, or config. Each entry has the following fields:

- **path** (required, string): The file path or directory path where the component lives. Relative to the repo root.
- **description** (required, string): A single short sentence stating the component's purpose and role. Do not include method names, route lists, interface names, or error types. Details like these churn every time the component grows. A good description should survive multiple features being added without needing an edit.
- **orchestration** (optional, boolean): Whether the component is an [orchestration module](glossary.md#orchestration-module) rather than an [implementation module](glossary.md#implementation-module). Defaults to `false` if omitted.

## Example

```yaml
- path: cmd/myapp
  description: Wires real dependencies into the application and starts the server.
  orchestration: true

- path: internal/orders
  description: Orchestrates order placement, fulfillment, and cancellation workflows.
  orchestration: true

- path: internal/postgres
  description: PostgreSQL-backed implementations of the domain repository interfaces.

- path: internal/httpapi
  description: HTTP handlers that translate requests into domain calls and format responses.

- path: deploy/containers
  description: Container images and runtime configuration for deploying the application.
```

## How the Architecture Is Used

The architecture document is the component map. It records what exists, so write it after the code and update it as the code changes. Before writing code, read it to find where the code belongs ([Architecture Guidelines](architecture.md)).

Keeping it accurate matters more than keeping it complete. A component whose description no longer matches what it does is worse than a component that is missing.
