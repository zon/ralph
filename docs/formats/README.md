# File Formats

Documentation of file formats used in Ralph development.

## Directory Structure

```
/specs/
├── architecture.yaml
└── features/
    └── <component>/
        └── <feature>/
            ├── spec.md
            ├── flow.md
            └── architecture.yaml
```

The top-level `/specs/architecture.yaml` covers the **current** modules of the application ([Architecture Format](./architecture.md)).

Specs, flows, and per-feature architecture are co-located under `/specs/features`:

- `spec.md` — behavioral requirements and scenarios ([Spec Format](./specs.md))
- `flow.md` — idealized domain logic ([Flow Format](./flows.md))
- `architecture.yaml` (optional) — **future** modules introduced by this feature ([Architecture Format](./architecture.md))

A **component** is a top-level deployment or ownership boundary — a distinct service, app, or library that could be developed and deployed independently. Good component names reflect runtime identity (`api`, `worker`, `frontend`), not internal organization.

A **feature** is a coherent slice of user-facing or system-facing behavior — something a user can do, or something the system does on their behalf. Good feature names describe what the system does (`auth`, `payments`, `notifications`), not how it does it (`jwt-handler`, `stripe-client`). If a feature grows too large to read comfortably, split it by sub-feature rather than by implementation detail.

## Formats

### [Specs](./specs.md)

The spec format for describing system behavior using structured requirements and scenarios.

### [Flows](./flows.md)

The flow format for documenting idealized domain logic as implementation contracts.

### [Architecture](./architecture.md)

The architecture format for outlining deep modules in YAML.