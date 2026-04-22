# Analyze Repo and Generate Specs

You are a software architect analyzing a codebase to produce domain specs.

## Task

Analyze the repository and produce a `/specs` directory of behavior-first spec files covering each domain area of the system.

## Context

Spec files follow the format defined in `docs/planning/specs.md`. Each spec covers one logical domain: observable behavior, inputs/outputs, and error conditions — not implementation details.

## Instructions

1. Read `docs/planning/specs.md` to internalize the spec format and the distinction between requirements, scenarios, and implementation detail before writing anything.

2. Orient yourself in the repository:
   - Read `CLAUDE.md` and `README.md` (if present) for purpose and top-level concepts
   - Read any manifest files (`go.mod`, `package.json`, `pyproject.toml`, etc.) for module name and dependencies
   - List the repo root directory

3. Identify the distinct deliverables:
   - Scan the entry-point directory (`cmd/`, `bin/`, `apps/`, or equivalent) for individual binaries or services
   - Check for deployment artifacts: `Dockerfile`, `Containerfile`, Helm charts, `docker-compose.yml`, Kubernetes manifests
   - For each deliverable, note its type: CLI tool, HTTP service, background worker, library, etc.
   - This step determines the organization pattern in step 5 — do not skip it

4. For each deliverable identified in step 3, enumerate its user-visible surface:
   - CLI tool: list every subcommand and flag group
   - HTTP service: list every route, webhook event type, or API endpoint
   - Worker: list every trigger, queue, or scheduled event it handles
   - Then read the relevant source files to understand the behavior, inputs, outputs, and error conditions for each entry point

5. Choose a spec organization pattern based on what step 3 found, then group your findings accordingly:
   - **Single deliverable** → organize by feature area (one spec per major capability or command group)
   - **Multiple deliverables of different types** (e.g., a CLI tool + a web service) → organize by component: one spec file per deliverable, scoped to its user-visible features; add shared cross-cutting concerns (config, auth) as separate specs only if they have their own observable behavior
   - **Multiple services in a distributed system** → organize by bounded context

   Aim for 4–8 spec files total. Prefer fewer, broader specs over many narrow ones.

6. For each spec, draft the file using the format from `docs/planning/specs.md`:
   - `## Purpose` — one paragraph describing the domain
   - `### Requirement:` sections each with a SHALL/MUST/SHOULD statement
   - `#### Scenario:` blocks using GIVEN/WHEN/THEN for both happy path and error conditions
   - Keep requirements behavior-focused; omit implementation detail

7. Write each spec file to `/specs/<domain>.md`. Create the `/specs` directory at the repo root if it does not exist.

8. After writing all files, list the files created, state which organization pattern was chosen and why, and note any areas where behavior was ambiguous or could not be fully inferred from the source.

## Output

A `/specs` directory at the repo root containing one Markdown spec file per domain area, formatted according to `docs/planning/specs.md`. No implementation details, no internal class or function names, no library choices.
