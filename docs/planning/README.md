# Development

The planning and development workflow followed by Ralph.

## Directory Structure

```
/specs/
├── features/
│   └── <component>/
│       └── <feature>/
│           ├── spec.md
│           └── flow.md
└── modules/
    └── <module>.md
```

### Features

Specs and flows are co-located under `/specs/features`:

- `spec.md` — behavioral requirements and scenarios ([Spec Format](./specs.md#spec-format))
- `flow.md` — idealized domain logic ([Flow Format](./flows.md#flow-file-format))

A **component** is a top-level deployment or ownership boundary — a distinct service, app, or library that could be developed and deployed independently. Good component names reflect runtime identity (`api`, `worker`, `frontend`), not internal organization.

A **feature** is a coherent slice of user-facing or system-facing behavior — something a user can do, or something the system does on their behalf. Good feature names describe what the system does (`auth`, `payments`, `notifications`), not how it does it (`jwt-handler`, `stripe-client`). If a feature grows too large to read comfortably, split it by sub-feature rather than by implementation detail.

When the repo has a single component, omit the component directory:

```
/specs/features/<feature>/
├── spec.md
└── flow.md
```

### Modules

Module designs live under `/specs/modules/<module>.md`. A **module** is a deep implementation unit — a self-contained package or subsystem with a well-defined interface. Module docs describe internal structure, key invariants, and design rationale that isn't captured by feature specs or flows.

## Specs

[Specs](./specs.md) describe system behavior using structured requirements and scenarios.

## Architecture

### Flow

[Flows](./flows.md) document idealized high level domain logic designs.

### Modules

[Modules](./modules.md) describe system architecture as a series of deep code modules.

## Projects

[Projects](./projects.md) contain coding instruction for agents developing the system.