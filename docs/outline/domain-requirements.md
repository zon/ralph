# Outline: Domain Requirements

## What are Domain Requirements

Domain requirements capture the business or domain rules the system must enforce — not how the system works internally, but what it must do and why. They document the problem being solved, the constraints that shaped decisions, and the language of the domain.

## How to Write Requirements

1. Identify the problem the system is meant to solve and who it solves it for.
2. Define the core domain concepts before writing any requirements.
3. Write requirements top-down: start with the most fundamental rules, then add constraints and edge cases.
4. For each requirement, be explicit about what is out of scope — unstated assumptions become future bugs.

## Output Format

Create one file per domain area under `docs/domain-requirements/`. Also create `docs/domain-requirements/README.md` with a link to each file.

### README.md

```markdown
# Domain Requirements

- [<Domain Area>](<filename>.md) — <one-line description>
```

### Per-feature file

Each file must begin with a glossary of terms specific to that domain area, followed by one section per requirement.

#### Glossary

```markdown
## Glossary

- **<Term>:** <Definition — what this means in the context of this system.>
```

#### Requirements

```markdown
## <Requirement Name>

**Goal:** <What the system must do, in one sentence.>

**Why:** <The business or domain reason this requirement exists.>

**Acceptance criteria:**
- <Observable behavior that confirms the requirement is met>

**Out of scope:** <What this requirement explicitly does not cover.>
```

Each acceptance criterion must describe observable behavior — not an implementation detail. Write from the perspective of what the system enforces or guarantees.

**Good acceptance criteria examples:**
- A project is rejected if it references a step name that does not exist
- An agent cannot be assigned to a task it does not have permission to execute
- A run is marked failed if any required step does not complete within its timeout

**Bad acceptance criteria examples:**
- Calls `Validate()` on the project struct
- Returns HTTP 400 on invalid input
- Logs the error to stderr
