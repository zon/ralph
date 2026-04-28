---
name: plan-spec
description: Creates a spec document in /specs following the project's OpenSpec format. Use when the user wants to write a spec, plan a feature spec, document behavior requirements, or add scenarios for a feature area.
---

# Plan Spec

Create a well-formed spec file in `./specs/` based on the user's description of the feature or behavior area.

## Steps

1. **Understand the scope.** If the user's request is vague, ask clarifying questions:
   - What feature or behavior area does this spec cover?
   - Which component does it belong to? (e.g. `ralph`, `webhook`)
   - Is this a new feature or documenting existing behavior?
   - Are there known edge cases or failure modes to capture?

2. **Read the spec format docs** to refresh your understanding:
   - `docs/planning/specs.md`

3. **Determine the file path.** Check the existing `specs/` structure to match its convention:
   - Single component repo: `specs/<feature>.md`
   - Multi-component repo: `specs/<component>/<feature>.md`
   - Feature names are lowercase, hyphen-separated, and describe what the system does (`auth`, `payments`) — not how it does it (`jwt-handler`, `stripe-client`)
   - Check existing files under `specs/` to pick a consistent name and avoid duplication

4. **Choose the rigor level** (default to Lite):
   - **Lite** — short behavior-first requirements, clear scope and non-goals, a few concrete acceptance checks. Use for most changes.
   - **Full** — use only for cross-component changes, API/contract changes, migrations, or security/privacy concerns where ambiguity could cause expensive rework.

5. **Draft the spec.** Follow the format exactly:
   - `## Purpose` — one or two sentences describing the domain
   - `### Requirement: <Name>` — a specific behavior the system MUST/SHALL/SHOULD have; use RFC 2119 keywords
   - `#### Scenario: <Name>` — GIVEN / WHEN / THEN / AND steps for each requirement; cover happy path and key edge cases

   Keep spec content behavior-facing:
   - Observable inputs, outputs, and error conditions
   - External constraints (security, reliability, compatibility)
   - Do NOT include class/function names, library choices, or implementation steps

6. **Write the file** to `./specs/<component>/<feature>.md`.

7. **Report** the file path and a one-line summary of what the spec covers.

## Output Format

```markdown
# <Feature> Specification

## Purpose

<One or two sentences describing the domain this spec covers.>

## Requirements

### Requirement: <Requirement Name>

The system SHALL/MUST/SHOULD <observable behavior>.

#### Scenario: <Happy path name>

- GIVEN <precondition>
- WHEN <action>
- THEN <expected outcome>
- AND <additional assertion>

#### Scenario: <Edge case name>

- GIVEN <precondition>
- WHEN <action>
- THEN <expected outcome>

### Requirement: <Another Requirement>

...
```

## RFC 2119 Keyword Guide

| Keyword | Use when |
|---------|----------|
| MUST / SHALL | Absolute requirement — no exceptions |
| SHOULD | Recommended; exceptions are valid but must be justified |
| MAY | Optional behavior |

## Notes

- One spec file per feature area. If a feature grows too large, split by sub-feature, not by implementation detail.
- Scenarios must be testable — if you cannot imagine writing an automated test for it, rewrite the scenario.
- Non-goals are valuable: if something is out of scope, say so explicitly.
- Avoid documenting the "how" — that belongs in `/designs` or `/projects`.
