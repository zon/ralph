# Non-Orchestration Module Review Plan

Review every module outside the `orchestration` category against the standards
in [docs/code.md](docs/code.md) and [docs/testing.md](docs/testing.md), using
the `ralph-review-module` skill, and report or fix any gaps found.

## Implementation

- [x] `internal/workflow`
- [x] `internal/ai`
- [x] `internal/opencode`
- [x] `internal/argo`
- [x] `internal/k8s`
- [x] `internal/git`
- [x] `internal/workspace`
- [x] `internal/github`
- [x] `internal/config`
- [x] `internal/webhookconfig`
- [x] `internal/notify`
- [x] `internal/version`
- [x] `internal/docker`
- [x] `internal/testutil`

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
