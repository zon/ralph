# Non-Orchestration Module Review Plan

Review every module outside the `orchestration` category against the standards
in [docs/code.md](docs/code.md) and [docs/testing.md](docs/testing.md), using
the `ralph-review-module` skill, and report or fix any gaps found.

## Implementation

- [ ] `internal/workflow`
- [ ] `internal/ai`
- [ ] `internal/opencode`
- [ ] `internal/argo`
- [ ] `internal/k8s`
- [ ] `internal/git`
- [ ] `internal/workspace`
- [ ] `internal/github`
- [ ] `internal/config`
- [ ] `internal/webhookconfig`
- [ ] `internal/notify`
- [ ] `internal/version`
- [ ] `internal/docker`
- [ ] `internal/testutil`

## Pipeline

- [ ] `internal/context`
- [ ] `internal/output`

## Lifecycle

- [ ] `internal/cleanup`

## Entry

Reviewed last — entry modules wire everything else, so review them once their
dependencies' compliance is known.

- [ ] `cmd/ralph`
- [ ] `cmd/ralph-webhook`
- [ ] `internal/cmd`
