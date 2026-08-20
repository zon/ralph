# Orchestration Pattern

Orchestration is a pattern for structuring domain logic. An orchestration sequences steps, enforces domain conditions, and delegates the work to other modules.

A coding agent writes the orchestration code while implementing a feature. Decide during planning whether a feature needs orchestration.

## Writing Orchestrations

Write orchestration code in the feature's implementation language:

- **Pass failures through.** Propagate them from helpers directly using the language's idiomatic mechanism (returned errors, thrown exceptions, result types). Only introduce a named error value when it represents a distinct domain condition with no underlying cause (e.g. `CartError.Empty` is a state, not a failure).
- **No debug code.** Remove all logger calls, debug statements, and diagnostic output.
- **Delegate side effects.** Hand off database writes, network calls, and notifications to helpers rather than performing them in the orchestration body. How helpers are wired is the repo's policy.
- **No infrastructure types.** Use domain nouns, not framework types like request contexts or HTTP writers.
- **Only write bodies that are pure orchestration.** Every line must be a domain condition, a named step call, or a return value. If writing the body would require literals, string construction, or format details, don't write it. Just call the function by name.

Keep an orchestration short enough to read in one pass, typically under 20 lines. If it grows longer, split it into named sub-orchestrations.

## Testing Orchestrations

Tests cover orchestration decisions only. Every line in a test body should be a domain-language call: setup, invocation, or assertion. If an assertion requires a literal value, a file path, a URL, or any format detail, extract it into a named test helper. Each test verifies one domain outcome: given this domain state, calling the orchestration produces this domain result.

## Example

An orchestration and the test that covers its decisions, in TypeScript:

```typescript
class Checkout {
    constructor(
        private orders: OrdersClient,
        private payments: PaymentsClient,
        private email: EmailClient,
    ) {}

    async checkout(cart: Cart, user: User) {
        if (cart.isEmpty()) return CartError.Empty

        const order = this.orders.create(cart, user)

        if (!await this.payments.charge(order)) {
            this.orders.cancel(order)
            await this.email.sendDeclined(order, user)
            return
        }

        await this.email.sendConfirmation(order, user)
        return order
    }
}
```

```typescript
test("successful checkout", () => {
    const user = users.any()
    const basket = cart.any().withItems(cart.anItem())
    const svc = checkout.forTest()
    const order = svc.checkout(basket, user)
    expect(order).toMatchOrder(basket, user)
    expect(email.sent()).toContain(email.confirmation(order, user))
})

test("payment declined", () => {
    const user = users.any()
    const basket = cart.any().withItems(cart.anItem())
    const svc = checkout.forTest({ payments: payments.thatDeclines() })
    svc.checkout(basket, user)
    expect(orders.created()).toBeEmpty()
    expect(email.sent()).toContain(email.declined(user))
})
```

## Module Structure

The orchestration function lives in an [orchestration module](glossary.md#orchestration-module). Each helper lives in an [implementation module](glossary.md#implementation-module).

Test helpers such as HTTP client wiring, fixture builders, and assertion utilities belong in or beside their implementation modules, never in the orchestration module.

Fixture builders for input types (e.g. a struct the caller passes into the orchestration) belong with the module that owns the type.

Record orchestration modules in `specs/architecture.yaml`. See [Architecture Format](architecture-outline.md).

## What Orchestrations Are Not

- **Not a spec.** Orchestrations do not define behavioral guarantees. Put those in a [spec](specs.md).
- **Not a branching tree.** Orchestrations should be exhaustive but minimize paths. If an orchestration has many branches, that is a signal to simplify the design, not to add more cases.
