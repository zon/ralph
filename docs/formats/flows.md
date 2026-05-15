# Flow Format

The flow format defines implementation contracts using idealized code. A flow document specifies function names, orchestration structure, and helper vocabulary that the real implementation must preserve. Flows omit anything that would obscure the logic: no error wrapping, no debug code, no retries, no dependency injection boilerplate.

Flow functions are implemented as written — same names, same signatures, same orchestration structure.

## Structure

### Writing Flows

Write in the language the feature is implemented in, using an idealized dialect:

- **Pass failures through.** Propagate failures from helpers directly using the language's idiomatic mechanism (returned errors, thrown exceptions, result types). Only introduce a named error value when it represents a distinct domain condition with no underlying cause (e.g. `CartError.Empty` is a state, not a failure).
- **No debug code.** Remove all logger calls, debug statements, and diagnostic output.
- **Use dependency injection for side effects.** Any helper that performs side effects (database writes, network calls, notifications) must be injected rather than called as a static function. Declare the flow as a method on a struct whose fields are the required interfaces; the caller wires in the real implementations, and tests substitute mocks.
- **No infrastructure types.** Use domain nouns, not framework types like request contexts, HTTP writers, etc.
- **Only write bodies that are pure orchestration.** Every line must be a domain condition, a named step call, or a return value. If writing the body would require literals, string construction, or format details, don't write it — just call the function by name.

A flow should be short enough to read in one pass — typically under 20 lines. If it grows longer, split it into named sub-flows.

### Writing Tests

Tests cover orchestration decisions only — that the right helpers were called under the right conditions. Every line in a test body should be a domain-language call: setup, invocation, or assertion. If an assertion requires a literal value, a file path, a URL, or any format detail, extract it into a named test helper. Each test covers one domain outcome: given this domain state, calling the flow produces this domain result.

## File Format

````markdown
# Checkout Flow

## Purpose
Process a cart purchase and notify the user.

## Flow

**Module:** `checkout`

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

### Helpers

- **`orders.create(cart, user)`** — builds an order record from the cart contents and user identity
- **`payments.charge(order)`** — attempts to collect payment for the order; returns whether it succeeded
- **`orders.cancel(order)`** — marks the order as void so it is never fulfilled
- **`email.sendConfirmation(order, user)`** — notifies the user that their purchase was successful
- **`email.sendDeclined(order, user)`** — notifies the user that their payment was not accepted

## Tests

**Module:** `checkout`

```typescript
test("successful checkout", () => {
    const user = users.any()
    const basket = cart.any().withItems(cart.anItem())
    const svc = checkout.withMocks()
    const order = svc.checkout(basket, user)
    expect(order).toMatchOrder(basket, user)
    expect(email.sent()).toContain(email.confirmation(order, user))
})

test("payment declined", () => {
    const user = users.any()
    const basket = cart.any().withItems(cart.anItem())
    const svc = checkout.withMocks({ payments: payments.thatDeclines() })
    svc.checkout(basket, user)
    expect(orders.created()).toBeEmpty()
    expect(email.sent()).toContain(email.declined(user))
})

test("empty cart", () => {
    const svc = checkout.withMocks()
    const result = svc.checkout(cart.empty(), users.any())
    expect(result).toBe(CartError.Empty)
})
```

### Helpers

- **`checkout.withMocks(overrides?)`** — constructs a `Checkout` with default mock implementations; pass overrides to substitute specific clients
- **`users.any()`** — returns a valid user in a default state suitable for checkout
- **`cart.any()`** — returns a cart builder; call `.withItems(...)` to populate it
- **`cart.anItem()`** — returns a purchasable item with a non-zero price
- **`cart.empty()`** — returns a cart with no items
- **`payments.thatDeclines()`** — returns a payments mock whose next charge attempt fails
- **`orders.created()`** — returns the list of orders persisted during the test
- **`email.sent()`** — returns the list of emails sent during the test
- **`email.confirmation(order, user)`** — constructs the expected confirmation email value for assertion
- **`email.declined(user)`** — constructs the expected declined-payment email value for assertion
````

| Element | Purpose |
|---------|---------|
| `## Purpose` | One sentence describing what the flow accomplishes |
| `## Flow` | The implementation contract — function names, signatures, and orchestration are fixed |
| `## Flow > **Module**` | The module where the flow function is implemented |
| `## Flow > ### Helpers` | One line per helper called in the flow, written as the qualified call site (`module.function(args)`), with a description of its domain role |
| `## Tests` | The test contract — test names, bodies, and helper vocabulary the implementation must match |
| `## Tests > **Module**` | The module where the tests are implemented |
| `## Tests > ### Helpers` | One line per test helper, written as the qualified call site (`module.function(args)`), with a description of what domain state it sets up or asserts |

## Module Structure

The flow function lives in an [orchestration module](../glossary.md#orchestration-module). Each helper lives in an [implementation module](../glossary.md#implementation-module).

Every flow document must declare its module assignments. The `**Module:**` line under `## Flow` names the orchestration module; the `**Module:**` line under `## Tests` names the test module. These are implementation contracts — the code must match.

Tests live in or beside the module they test, using whatever convention the implementation language idiomatically uses (e.g. Go's `_test.go` files in the same package, Rust's inline `#[cfg(test)] mod tests`, a sibling `*.test.ts` file, a parallel `test/` tree). The test module name in the document should match the module under test; choose a distinct name only if the language enforces one (e.g. a Go external `foo_test` package). A test helper that asserts on payment behavior belongs near the module it tests, not in a generic test module.

Orchestration modules must not contain helper methods that perform or test implementation details. Test helpers — mock factories, HTTP client wiring, fixture builders, assertion utilities — belong in or beside the implementation modules they serve. The orchestration module's test file contains only test functions that exercise orchestration logic; it calls helpers imported from implementation modules, never defines them. If a helper is needed to set up or assert on an implementation concern, it lives with that concern.

## File Location

See [Directory Structure](./README.md#directory-structure) for where flow files are located.

## What Flows Are Not

- **Not a spec.** Flows do not define behavioral guarantees. Put those in `/specs`.
- **Not a branching tree.** Flows should be exhaustive but designed to minimize paths. If a flow has many branches, that is a signal to simplify the design, not to add more cases.
