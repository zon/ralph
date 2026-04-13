# webhook/structure: handleWebhook mixes domain logic with infrastructure

**Module:** `internal/webhook`
**Concern:** Structure

## Issues

- `internal/webhook/server.go:55` - `handleWebhook` is not a pure domain function; it embeds infrastructure details throughout its implementation:
  1. **HTTP semantics** (lines 58, 65, 73, 80, 93, 103, 108) - `c.JSON()`, `c.GetHeader()`, `c.Status()` are gin-specific
  2. **Logging infrastructure** (lines 68, 72, 79, 88, 92, 98, 102, 130, 133, 138, 141) - `logger.Verbosef()` calls are infrastructure, not domain
  3. **Goroutine management** (line 107) - `go submitWorkflow(...)` is an implementation detail (concurrency model)
  4. **Crypto operations** (lines 148-163) - `validateSignature` with HMAC-SHA256 is infrastructure

The domain logic is: "receive webhook → validate → filter → event → workflow → submit." The current implementation mixes this with HTTP request parsing, logging calls, and async goroutine launch.

Per the domain functions standard, `handleWebhook` should orchestrate domain steps without implementation details. Extract infrastructure concerns so the function reads as a pure domain process.

## Recommended Refactoring

1. Extract HTTP layer: create a thin adapter that calls a domain handler with parsed data
2. Extract logging: domain functions should not call logger directly; logging is observability infrastructure
3. Extract async submission: the caller should handle concurrency, not `handleWebhook`
4. Extract signature validation: this is security infrastructure that should be handled by the HTTP layer before domain logic