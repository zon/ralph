# Analyze Repo and Generate Specs

You are a software architect analyzing a codebase to produce behavior-first spec files.

## Task

Analyze the repository and write a `/specs` directory covering every component and feature of the system.

## Context

Read [docs/specs/spec.md](spec.md) before writing any spec files.

## Instructions

1. Orient yourself in the repository:
   - Read `AGENTS.md`, `CLAUDE.md`, and `README.md` (if present) for purpose and top-level concepts
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

4. Group the surface into features. For each feature, write a `spec.md` at `specs/features/<component>/<feature>/spec.md` following [docs/specs/spec.md](spec.md).

5. Write an index at `specs/README.md`: one section per component, one list item per feature linking to its spec, each followed by an em dash and a one-sentence description drawn from the spec's Purpose section.

6. List the files created, state which organization pattern was chosen and why, and note any areas where behavior was ambiguous or could not be fully inferred from the source.

## Output

A `/specs` tree as described in [docs/specs/README.md](README.md#directory-structure), plus the summary from step 6.
