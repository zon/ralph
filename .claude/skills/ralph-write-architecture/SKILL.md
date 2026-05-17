---
name: ralph-write-architecture
description: Creates or edits the architecture document at specs/architecture.yaml. Use when the user wants to outline the deep modules of an application, document current architecture, or plan future modules.
---

# Write Architecture

Create or update an architecture document. Architecture files live in two places:

- **`./specs/architecture.yaml`** — current modules of the application.
- **`./specs/features/<component>/<feature>/architecture.yaml`** (optional) — future modules introduced by a specific feature.

## Steps

1. **Read the architecture format docs** at [docs/formats/architecture.md](docs/formats/architecture.md).

2. **Determine the target file:**
   - If documenting **current** architecture, use `./specs/architecture.yaml`.
   - If planning **future** modules for a specific feature, use `./specs/features/<component>/<feature>/architecture.yaml`. Ask the user for the component and feature names if unclear.

3. **Read the existing architecture document** at the target path if one exists, so edits preserve unrelated modules.

4. **Clarify the scope.** If the user's request is vague, ask clarifying questions before proceeding.

5. **Read the spec and orchestration** when architecting a feature (`specs/features/<component>/<feature>/spec.md` and `orchestration.md`) to identify what modules are needed and how responsibilities divide between them.

6. **Check helper namespaces.** Examine both the `## Orchestration > ### Helpers` and `## Tests > ### Helpers` sections. Each namespace (e.g. `orders.*`, `target.*`, `source.*`) implies a module home. Add a module entry for any namespace that lives outside the modules already listed.

7. **Flag shared input types.** When a caller module (e.g. `cmd`) and an orchestration module both appear, identify any type that both sides need and ensure it has a clear module home. Add a module entry for any type that would otherwise be claimed by both sides.

8. **Survey the codebase** when documenting current architecture to confirm module paths and responsibilities.

9. **Draft the architecture** following the format in [docs/formats/architecture.md](docs/formats/architecture.md).

10. **Write the file** to the target path.

11. **Report** the file path and a one-line summary of the modules covered.
