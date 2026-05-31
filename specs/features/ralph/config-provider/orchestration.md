# Config Provider Orchestration

## Purpose

Collect an API key for a single AI provider, merge it into `.ralph/auth.yaml`, and sync the full credentials file to the Kubernetes Secret in the ralph namespace.

## Orchestration

**Module:** `internal/orchestration/configprovider`

```go
type Runner struct {
    prompt PromptClient
    auth   AuthClient
    k8s    K8sClient
}

func (r *Runner) Run(ctx context.Context, provider, kubeContext, namespace string) error {
    key, err := r.prompt.ProviderKey(provider)
    if err != nil {
        return err
    }
    existing, err := r.auth.Load()
    if err != nil {
        return err
    }
    existing[provider] = key
    if err := r.auth.Write(existing); err != nil {
        return err
    }
    return r.k8s.StoreProviderSecret(ctx, kubeContext, namespace, existing)
}
```

### Helpers

- **`r.prompt.ProviderKey(provider)`** — prompts the user for the API key for the named provider; returns an error if the user enters a blank value
- **`r.auth.Load()`** — reads `.ralph/auth.yaml` and returns existing provider keys as a map; returns an empty map if the file does not exist
- **`r.auth.Write(keys)`** — writes the full provider key map to `.ralph/auth.yaml`
- **`r.k8s.StoreProviderSecret(ctx, kubeContext, namespace, keys)`** — creates or updates the `provider-credentials` Kubernetes Secret in the specified namespace with the full provider key map

## Tests

**Module:** `internal/orchestration/configprovider`

```go
test("key written and secret updated", func(t) {
    r := runner.withMocks(prompt.returning("sk-ant-123"))
    r.Run(ctx, "anthropic", kubeContext, namespace)
    assert(auth.written()).equals(auth.keys("anthropic", "sk-ant-123"))
    assert(k8s.storedProviderSecret()).equals(auth.keys("anthropic", "sk-ant-123"))
})

test("existing provider keys preserved", func(t) {
    r := runner.withMocks(
        auth.containing(auth.keys("google", "AIza-existing")),
        prompt.returning("sk-ant-123"),
    )
    r.Run(ctx, "anthropic", kubeContext, namespace)
    assert(auth.written()).equals(auth.keys("anthropic", "sk-ant-123", "google", "AIza-existing"))
    assert(k8s.storedProviderSecret()).equals(auth.keys("anthropic", "sk-ant-123", "google", "AIza-existing"))
})

test("blank key returns error", func(t) {
    r := runner.withMocks(prompt.returningBlank())
    err := r.Run(ctx, "anthropic", kubeContext, namespace)
    assert(err).isNotNil()
    assert(auth.written()).isEmpty()
})
```

### Helpers

- **`runner.withMocks(...overrides)`** — constructs a `Runner` with default mock implementations; pass override mocks to substitute specific clients
- **`auth.empty()`** — returns an `AuthClient` mock that reports no existing keys
- **`auth.containing(keys)`** — returns an `AuthClient` mock that returns the given keys from `Load`
- **`auth.keys(pairs...)`** — builds a `map[string]string` from alternating provider name / key string pairs
- **`auth.written()`** — returns the keys passed to `Write` on the mock `AuthClient`; nil if `Write` was not called
- **`prompt.returning(key)`** — returns a `PromptClient` mock whose `ProviderKey` returns the given key
- **`prompt.returningBlank()`** — returns a `PromptClient` mock whose `ProviderKey` returns an error for a blank entry
- **`k8s.storedProviderSecret()`** — returns the keys passed to `StoreProviderSecret` on the mock `K8sClient`
