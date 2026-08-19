# Architecture Guidelines

## Before You Start

Read `specs/architecture.yaml` before writing any code if it exists. See [Architecture Format](architecture-outline.md).

## Component Placement

Every piece of code belongs in a specific component. Before writing, ask:

1. **Does an existing component own this concern?** Check `specs/architecture.yaml`. If a component already covers the concern, add the code there rather than duplicating logic.
2. **Does the orchestration pattern assign this code to a component type?** Coordination logic belongs in an [orchestration module](glossary.md#orchestration-module). Low-level work belongs in an [implementation module](glossary.md#implementation-module). See [Orchestration Pattern](orchestration.md).
3. **Is there no existing home?** If neither applies, determine whether it belongs in an existing component (by expanding its scope) or in a new one. If a new component is needed, add it to `specs/architecture.yaml` once the code is written.

## Component Types

Each component is either an [implementation module](glossary.md#implementation-module) or, when `specs/architecture.yaml` sets `orchestration: true`, an [orchestration module](glossary.md#orchestration-module). A component with no code is neither.

### Orchestration Modules

- A small app typically has one.
- Do not add code to an orchestration module unless its orchestration calls it.
- If an orchestration module is accumulating logic that is not pure coordination, move that logic into an implementation module.
- Split a growing orchestration module along deep concern boundaries.

### Implementation Modules

- Keep each component focused on its declared concern. Do not extend a component's scope without updating `specs/architecture.yaml`.
- Prefer deepening an existing component over creating a new one for the same concern.
- Expose only what callers need. Keep internal details unexported.
- If a component meant to hold only pure functions accumulates side-effectful code, move that code into a different implementation module.

## Related

- [Glossary](glossary.md) — definitions of deep, pure, orchestration, and implementation modules
