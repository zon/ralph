# Analyze Repo and Generate Specs

You are a software architect analyzing a codebase to produce domain specs.

## Task

Analyze the repository and produce a `/specs` directory of behavior-first spec files covering each domain area of the system.

## Context

Spec files follow the format defined in `docs/planning/specs.md`. Each spec covers one logical domain: observable behavior, inputs/outputs, and error conditions — not implementation details.

## Instructions

1. Read `docs/planning/specs.md` to internalize the spec format and the distinction between requirements, scenarios, and implementation detail before writing anything.

2. Map the domain boundaries of the system by reading:
   - `CLAUDE.md` for high-level orientation
   - `README.md` (if present) for purpose and top-level concepts
   - Any manifest files (`go.mod`, `package.json`, `pyproject.toml`, etc.) for the module name and dependencies
   - Directory listing of the repo root to identify major packages and components

3. For each top-level package or significant subdirectory, read the source files to understand what each component does, what it accepts as input, and what it produces or changes.

4. Group your findings into logical domain areas. Aim for 4–8 spec files organized by feature area, component, or bounded context as appropriate for the system.

5. For each domain area, draft a spec file using the format from `docs/planning/specs.md`:
   - `## Purpose` — one paragraph describing the domain
   - `### Requirement:` sections each with a SHALL/MUST/SHOULD statement
   - `#### Scenario:` blocks using GIVEN/WHEN/THEN for both happy path and error conditions
   - Keep requirements behavior-focused; omit implementation detail

6. Write each spec file to `/specs/<domain>.md`. Create the `/specs` directory at the repo root if it does not exist.

7. After writing all files, list the files created and note any domain areas where behavior was ambiguous or could not be fully inferred from the source.

## Output

A `/specs` directory at the repo root containing one Markdown spec file per domain area, formatted according to `docs/planning/specs.md`. No implementation details, no internal class or function names, no library choices.
