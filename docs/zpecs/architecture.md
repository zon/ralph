# Architecture Guidelines

## Before You Start

If it exists, read `specs/architecture.yaml` before writing code. See [Architecture Format](architecture-outline.md).

## Component Structure

Every piece of code belongs in a specific component. Before writing, ask whether an existing component owns the concern. If it does, add the code there rather than duplicating logic.

Keep the component set as small as it can be. Each component should earn its place by owning a distinct, substantial concern. Before adding, removing, or merging components, read the format guidance in [Architecture Format](architecture-outline.md).

### When to Create Components

Create a new component only for one of these reasons:

- **Separate orchestration from implementation.** Create one component for coordination and one for the low-level work it delegates. See [Orchestration Pattern](orchestration.md).
- **Separate major concerns.** Give each major concern its own [deep module](glossary.md#deep-module): a simple interface over a complex implementation.
- **Share common logic.** Move logic that several major concerns use into a shared component.

When none of these apply, add the code to an existing component.

### When to Remove Components

Look for a component whose only caller is one other component. When two components relate this way, combine them into a single deeper concern. The calling component absorbs the callee's work, or the callee folds into its only caller, whichever keeps the interface smaller.

## Component Types

`specs/architecture.yaml` sets each component's type: [implementation module](glossary.md#implementation-module) by default, [orchestration module](glossary.md#orchestration-module) when it sets `orchestration: true`. A component with no code is neither.

Coordination logic belongs in an [orchestration module](glossary.md#orchestration-module), low-level work in an [implementation module](glossary.md#implementation-module). See [Orchestration Pattern](orchestration.md).

### Orchestration Modules

- A small app typically has one.
- Do not add code to an orchestration module unless its orchestration calls it.
- If an orchestration module is accumulating logic that is not pure coordination, move that logic into an implementation module.
- Split a growing orchestration module along deep concern boundaries.

### Implementation Modules

- Keep each component focused on its declared concern. Do not extend a component's scope without updating `specs/architecture.yaml`.
- Prefer deepening an existing component over creating a new one for the same concern.
- Expose only what callers need.
- If a component meant to hold only pure functions accumulates side-effectful code, move that code into a different implementation module.
- Interfacing with CLI input parsers or HTTP hosting is usually an implementation concern.

## Related

- [Glossary](glossary.md) — definitions of deep, pure, orchestration, and implementation modules
