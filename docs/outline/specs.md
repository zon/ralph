# Analyze Repo and Generate Specs

You are a software architect analyzing a codebase to produce behavior-first spec files.

## Task

Analyze the repository and write a `/specs` directory covering every component and feature of the system.

## Context

Spec format, structure, organization patterns, and file sizing guidelines are defined in `docs/planning/specs.md`. Read it before doing anything else.

## Instructions

1. Orient yourself in the repository:
   - Read `CLAUDE.md` and `README.md` (if present) for purpose and top-level concepts
   - Read any manifest files (`go.mod`, `package.json`, `pyproject.toml`, etc.) for module name and dependencies
   - List the repo root directory

2. Identify the distinct components:
   - Scan the entry-point directory (`cmd/`, `bin/`, `apps/`, or equivalent) for individual binaries or services
   - Check for deployment artifacts: `Dockerfile`, `Containerfile`, Helm charts, `docker-compose.yml`, Kubernetes manifests
   - For each component, note its type: CLI tool, HTTP service, background worker, library, etc.

3. For each component, enumerate its user-visible surface:
   - CLI tool: list every subcommand and flag group
   - HTTP service: list every route, webhook event type, or API endpoint
   - Worker: list every trigger, queue, or scheduled event it handles
   - Read the relevant source files to understand behavior, inputs, outputs, and error conditions

4. Group the surface into features and write one spec file per feature, following the structure and format from `docs/planning/specs.md`.

5. List the files created, state which organization pattern was chosen and why, and note any areas where behavior was ambiguous or could not be fully inferred from the source.

## Output

A `/specs` directory at the repo root with one Markdown spec file per feature, organized and formatted according to `docs/planning/specs.md`.
