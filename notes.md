# Notes

## Project `code` entries not grounded in orchestration

When writing project requirements, agents (and humans) tend to invent `code` entries rather than sourcing them from the feature's `orchestration.md`. The result is improvised function signatures and struct shapes that don't reflect the intended design — the opposite of what the field is for.

The `code` field exists to relay orchestration details to the ralph agent so it implements the right shape. If the orchestration document has no code shape for a requirement, there is nothing to relay — the requirement should use `scenarios` and `items` only.

Fixes to consider:
- Strengthen the wording in `docs/formats/project.md` under the `code` field: make it explicit that entries must be sourced from `orchestration.md`, not composed freehand.
- Update the `ralph-write-project` skill to check the feature's `orchestration.md` before adding any `code` entry, and skip the field when no matching shape exists.
- Add a validation warning (or error) in `ralph validate` when a `code` entry appears in a requirement whose feature has no orchestration document.
