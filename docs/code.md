# Writing Code

## Before You Start

Read `specs/architecture.yaml` before writing any code — it is the module map for the entire codebase.

If you are working on a feature, also read its architecture file at `specs/features/<component>/<feature>/architecture.yaml` if one exists. A feature architecture file describes the modules introduced or modified by that feature: their paths, roles, and whether each is an orchestration or implementation module.

## Module Placement

Every piece of code belongs in a specific module. Before writing, ask:

1. **Does an existing module own this concern?** Check `specs/architecture.yaml`. If a module already covers the concern, add the code there rather than duplicating logic or creating a parallel path.
2. **Does an orchestration file assign this code to a specific module?** Orchestration files (`specs/features/<component>/<feature>/orchestration.md`) are implementation contracts — the `**Module:**` annotations are binding. Place the code in the module named by the orchestration.
3. **Is there no existing home?** If neither an existing module nor an orchestration file covers the concern, determine whether it belongs in an existing module (by expanding its scope) or requires a new module. If a new module is needed, update `specs/architecture.yaml` before writing the code.

## Orchestration Modules

Modules marked `orchestration: true` in `specs/architecture.yaml` are orchestration modules.

- Do not add code to an orchestration module that an orchestration file does not ask for. If an orchestration file does not call it, it does not belong there.
- Do not add implementation details — no string construction, no format literals, no I/O, no external calls, no helper utilities.
- If an orchestration module is accumulating logic that is not pure coordination, move that logic into an implementation module.

## Implementation Modules

- Keep each module focused on its declared concern. Do not extend a module's scope without updating `specs/architecture.yaml`.
- Prefer deepening an existing module over creating a new one for the same concern.
- Expose only what callers need; keep internal details unexported.
