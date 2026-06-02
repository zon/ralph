# Module Compliance Plan

## Review for compliance

- `internal/ai` — add Client interface, real implementation, and MockClient
- `internal/config` — add Client interface and MockClient (bare `Client struct{}` exists)
- `internal/webhookconfig` — introduce Client struct, interface, and MockClient

## Evaluate category

- `internal/logger` — determine whether implementation or pure
- `internal/cleanup` — determine whether implementation or pure
