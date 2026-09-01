# Orchestration Module Review Plan

Audit every module in the `orchestration` category against the standards in
[docs/zpecs/architecture.md](docs/zpecs/architecture.md) and [docs/testing.md](docs/testing.md),
and report or fix any gaps found.

## Modules

- [x] `internal/orchestration/run` — gaps found, project: `projects/remove-dead-prepare-execution.yaml`
- [x] `internal/orchestration/workspace`
- [x] `internal/orchestration/workflowrun` — gaps found, project: `projects/workflowrun-merge-conflict-ai-resolution.yaml`
- [x] `internal/orchestration/merge`
- [x] `internal/project` — gaps found, project: `projects/clean-up-project-helpers.yaml`
- [x] `internal/orchestration/validate`
- [x] `internal/orchestration/setremote`
- [x] `internal/orchestration/workflowtoken` — minor gofmt/dead-code cleanup noted, no project
- [x] `internal/services` — gaps found, project: `projects/services-implementation-category.yaml`
- [x] `internal/orchestration/argo`
- [x] `internal/orchestration/command` — gaps found, project: `projects/remove-dead-remote-command.yaml`
- [x] `internal/orchestration/pass` — removed: the `ralph pass` command is gone, since nothing writes completion into the project file

## Process per module

1. Run `ralph-review-module` with the module path as the argument.
2. Record the Compliant / Gaps / Recommendations summary below the module's
   checklist item.
3. If a project file was created for recommendations, note its path.
4. Check off the module once reviewed.
