# Models Orchestration

## Purpose

Load configured provider credentials, query each provider's API for available models, and print them in `provider/model-id` format — warning and continuing when a provider is unreachable.

## Orchestration

**Module:** `internal/orchestration/models`

```go
type Runner struct {
    auth     AuthClient
    provider ProviderClient
    output   OutputClient
}

func (r *Runner) Run(ctx context.Context, filter string) error {
    keys, err := r.auth.Load()
    if err != nil {
        return err
    }
    if len(keys) == 0 {
        return ErrNoProvidersConfigured
    }
    providers, err := filteredProviders(keys, filter)
    if err != nil {
        return err
    }
    for _, name := range providers {
        models, err := r.provider.ListModels(ctx, name, keys[name])
        if err != nil {
            r.output.Warn(name, err)
            continue
        }
        r.output.PrintModels(name, models)
    }
    return nil
}
```

### Helpers

- **`r.auth.Load()`** — reads `.ralph/auth.yaml` and returns existing provider keys as a map; returns an empty map if the file does not exist
- **`filteredProviders(keys, filter)`** — if `filter` is empty, returns all provider names present in `keys`; if `filter` names a recognised provider, returns just that name; if `filter` names an unrecognised provider, returns `ErrUnknownProvider`
- **`r.provider.ListModels(ctx, name, key)`** — queries the named provider's API using the given key and returns the available model IDs
- **`r.output.Warn(name, err)`** — prints a warning that the named provider could not be reached
- **`r.output.PrintModels(name, models)`** — prints each model ID to stdout in `provider/model-id` format, one per line

## Tests

**Module:** `internal/orchestration/models`

```go
test("models listed for each configured provider", func(t) {
    r := runner.withMocks(
        auth.containing(auth.keys("anthropic", "sk-ant", "google", "AIza")),
        provider.returning("anthropic", models.list("claude-opus-4-8", "claude-sonnet-4-6")),
        provider.returning("google", models.list("gemini-2.5-pro")),
    )
    r.Run(ctx, "")
    assert(output.printed()).equals(models.formatted("anthropic", "claude-opus-4-8", "claude-sonnet-4-6", "google", "gemini-2.5-pro"))
})

test("no providers configured — error returned", func(t) {
    r := runner.withMocks(auth.empty())
    err := r.Run(ctx, "")
    assert(err).isError(ErrNoProvidersConfigured)
})

test("provider API failure — warn and continue", func(t) {
    r := runner.withMocks(
        auth.containing(auth.keys("anthropic", "sk-ant", "google", "AIza")),
        provider.failing("anthropic"),
        provider.returning("google", models.list("gemini-2.5-pro")),
    )
    r.Run(ctx, "")
    assert(output.warned()).contains("anthropic")
    assert(output.printed()).equals(models.formatted("google", "gemini-2.5-pro"))
})

test("provider filter — only named provider queried", func(t) {
    r := runner.withMocks(
        auth.containing(auth.keys("anthropic", "sk-ant", "google", "AIza")),
        provider.returning("anthropic", models.list("claude-opus-4-8")),
    )
    r.Run(ctx, "anthropic")
    assert(provider.queried()).equals([]string{"anthropic"})
    assert(output.printed()).equals(models.formatted("anthropic", "claude-opus-4-8"))
})

test("unknown provider — error returned before any API call", func(t) {
    r := runner.withMocks(auth.containing(auth.keys("anthropic", "sk-ant")))
    err := r.Run(ctx, "foobar")
    assert(err).isError(ErrUnknownProvider)
    assert(provider.queried()).isEmpty()
})
```

### Helpers

- **`runner.withMocks(...overrides)`** — constructs a `Runner` with default mock implementations; pass override mocks to substitute specific clients
- **`auth.empty()`** — returns an `AuthClient` mock that reports no existing keys
- **`auth.containing(keys)`** — returns an `AuthClient` mock that returns the given keys from `Load`
- **`auth.keys(pairs...)`** — builds a `map[string]string` from alternating provider name / key string pairs
- **`provider.returning(name, models)`** — returns a `ProviderClient` mock that returns the given model list for the named provider
- **`provider.failing(name)`** — returns a `ProviderClient` mock that returns an error for the named provider
- **`provider.queried()`** — returns the list of provider names passed to `ListModels` on the mock `ProviderClient`
- **`models.list(ids...)`** — builds a `[]string` of model IDs
- **`models.formatted(pairs...)`** — builds the expected stdout output from alternating provider name / model ID pairs in `provider/model-id` format
- **`output.printed()`** — returns the lines written via `PrintModels` on the mock `OutputClient`
- **`output.warned()`** — returns the provider names passed to `Warn` on the mock `OutputClient`
