## Modules

Module docs describe the internal design of a deep implementation unit — a self-contained package or subsystem with a well-defined interface. They capture structure, invariants, and rationale that isn't visible from a feature spec or flow.

### What a Module Is

A module is a package or subsystem that:

- Has a well-defined interface other code calls
- Hides internal complexity behind that interface
- Maintains invariants that callers depend on but cannot verify from the outside
- Has design decisions that need explanation beyond what the code conveys

Not every package needs a module doc. Write one when the design contains non-obvious decisions, when the invariants are load-bearing and easy to accidentally break, or when the internal structure helps readers understand why the code is shaped the way it is.

### Where Modules Live

Module docs live under `/specs/modules/<module>.md`. See [Directory Structure](./README.md#directory-structure).

### Module File Format

````markdown
# Rate Limiter

## Purpose
Enforce per-key request quotas using a sliding window algorithm.

## Interface

```go
type Limiter struct { ... }

func New(window time.Duration, limit int) *Limiter
func (l *Limiter) Allow(key string) bool
```

## Invariants

- `Allow` is safe to call concurrently.
- At most `limit` calls with the same `key` return `true` within any `window`-length interval.
- State is in-memory and not shared across processes; limits are per-instance.

## Design

Each key maps to a circular buffer of timestamps. On each `Allow` call, the buffer drops entries older than `window`, then checks whether the remaining count is below `limit`. If so, it appends the current timestamp and returns `true`.

The buffer is pre-allocated at `limit` capacity so no allocations occur during steady-state operation. A mutex per key avoids global contention.

## Rationale

**Sliding window over fixed window.** Fixed windows allow a burst of 2× the limit across a window boundary. A sliding window eliminates this without requiring a token bucket, which would need a background goroutine to refill tokens.

**Per-key mutex over a single global lock.** A single lock serializes all keys. Per-key locking scales with the number of concurrent callers on distinct keys, which is the common case.

**No persistence.** Limits reset on restart. This is intentional: the rate limiter is a best-effort guard, not a billing-accurate counter. Persisting state would add complexity and a failure mode that the use case does not justify.
````

**Key elements:**

| Element | Purpose |
|---------|---------|
| `## Purpose` | One sentence describing what the module does |
| `## Interface` | The public types and functions other code calls |
| `## Invariants` | Behavioral guarantees the module maintains regardless of how it is used |
| `## Design` | Internal structure and how the module achieves its purpose |
| `## Rationale` | Non-obvious decisions and the tradeoffs that drove them |

`## Purpose` and `## Interface` are always required. Write the remaining sections only when the content is non-obvious — if a section would just restate what the code already says, omit it.

### What Module Docs Are Not

- **Not a spec.** Module docs describe implementation, not observable behavior. Put behavioral contracts in `/specs`.
- **Not a flow.** Flows describe domain orchestration. Module docs describe the internal structure of a subsystem — often the implementation of a helper the flow calls.
- **Not API docs.** Module docs explain *why* the interface is shaped the way it is. Per-function documentation belongs in the code itself.
- **Not exhaustive.** Cover the non-obvious. If the code is self-explanatory, skip the section.
