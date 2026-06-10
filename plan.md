# Orchestration Module Review Plan

Audit every module in the `orchestration` category against the standards in
[docs/code.md](docs/code.md) and [docs/testing.md](docs/testing.md), using
the `ralph-review-module` skill, and report or fix any gaps found.

## Modules

- [x] `internal/orchestration/run` — gaps found, project: `projects/remove-dead-prepare-execution.yaml`
- [x] `internal/orchestration/workspace`
- [x] `internal/orchestration/workflowrun` — gaps found, project: `projects/workflowrun-merge-conflict-ai-resolution.yaml`
- [x] `internal/orchestration/comment`
- [x] `internal/orchestration/merge`
- [x] `internal/project` — gaps found, project: `projects/clean-up-project-helpers.yaml`
- [x] `internal/orchestration/validate`
- [x] `internal/orchestration/setup`
- [x] `internal/orchestration/setconfig`
- [x] `internal/orchestration/webhooksetconfig`
- [x] `internal/orchestration/workflowtoken` — minor gofmt/dead-code cleanup noted, no project
- [x] `internal/webhook` — gaps found, project: `projects/webhook-orchestration-cleanup.yaml`
- [x] `internal/provisioning` — gaps found, project: `projects/provisioning-into-webhookconfig.yaml`
- [x] `internal/skills` — gaps found, project: `projects/skills-implementation-category.yaml`
- [x] `internal/architecture` — gaps found, project: `projects/architecture-remove-dead-schema.yaml`
- [x] `internal/services` — gaps found, project: `projects/services-implementation-category.yaml`
- [x] `internal/orchestration/argo`
- [x] `internal/orchestration/command` — gaps found, project: `projects/remove-dead-remote-command.yaml`
- [x] `internal/orchestration/pass` — gaps found, project: `projects/pass-confirmation-message.yaml`

## Process per module

1. Run `ralph-review-module` with the module path as the argument.
2. Record the Compliant / Gaps / Recommendations summary below the module's
   checklist item.
3. If a project file was created for recommendations, note its path.
4. Check off the module once reviewed.
