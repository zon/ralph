# Orchestration Module Review Plan

Audit every module in the `orchestration` category against the standards in
[docs/code.md](docs/code.md) and [docs/testing.md](docs/testing.md), using
the `ralph-review-module` skill, and report or fix any gaps found.

## Modules

- [ ] `internal/orchestration/run`
- [ ] `internal/orchestration/workspace`
- [ ] `internal/orchestration/workflowrun`
- [ ] `internal/orchestration/comment`
- [ ] `internal/orchestration/merge`
- [ ] `internal/project`
- [ ] `internal/orchestration/validate`
- [ ] `internal/orchestration/setup`
- [ ] `internal/orchestration/setconfig`
- [ ] `internal/orchestration/webhooksetconfig`
- [ ] `internal/orchestration/workflowtoken`
- [ ] `internal/webhook`
- [ ] `internal/provisioning`
- [ ] `internal/skills`
- [ ] `internal/architecture`
- [ ] `internal/services`
- [ ] `internal/orchestration/argo`
- [ ] `internal/orchestration/command`
- [ ] `internal/orchestration/pass`

## Process per module

1. Run `ralph-review-module` with the module path as the argument.
2. Record the Compliant / Gaps / Recommendations summary below the module's
   checklist item.
3. If a project file was created for recommendations, note its path.
4. Check off the module once reviewed.
