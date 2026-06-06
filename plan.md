# Orchestration Module Plan

Add orchestration modules so every entry point follows the entry → orchestration → implementation flow.

## Modules to create

- `internal/orchestration/validate` — project YAML validation via opencode
- `internal/orchestration/pass` — load, update, and save project requirement status
- `internal/orchestration/config/github` — GitHub App credential validation and K8s secret creation
- `internal/orchestration/config/opencode` — OpenCode auth file reading and K8s secret creation
- `internal/orchestration/config/pulumi` — Pulumi token resolution and K8s secret creation
- `internal/orchestration/config/webhook` — webhook configmap building, secret generation, and webhook registration
- `internal/orchestration/githubtoken` — configure GitHub App git authentication
