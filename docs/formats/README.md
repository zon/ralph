# File Formats

Documentation of file formats used in Ralph development.

## Directory Structure

```
/specs/
├── architecture.md
└── features/
    └── <component>/
        └── <feature>/
            ├── spec.md
            └── flow.md
```

The architecture doc lives at `/specs/architecture.md` ([Architecture Format](./architecture.md)).

Specs and flows are co-located under `/specs/features`:

- `spec.md` — behavioral requirements and scenarios ([Spec Format](./specs.md))
- `flow.md` — idealized domain logic ([Flow Format](./flows.md))

A **component** is a top-level deployment or ownership boundary — a distinct service, app, or library that could be developed and deployed independently. Good component names reflect runtime identity (`api`, `worker`, `frontend`), not internal organization.

A **feature** is a coherent slice of user-facing or system-facing behavior — something a user can do, or something the system does on their behalf. Good feature names describe what the system does (`auth`, `payments`, `notifications`), not how it does it (`jwt-handler`, `stripe-client`). If a feature grows too large to read comfortably, split it by sub-feature rather than by implementation detail.

When the repo has a single component, omit the component directory:

```
/specs/features/<feature>/
├── spec.md
└── flow.md
```

## Formats

### [Specs](./specs.md)

The spec format for describing system behavior using structured requirements and scenarios.

### [Flows](./flows.md)

The flow format for documenting idealized domain logic as implementation contracts.

### [Architecture](./architecture.md)

The architecture format for outlining deep modules in YAML.