## Flows

Flows are contracts on domain abstraction — idealized code that defines the function names, orchestration structure, and helper vocabulary the real implementation must preserve. A flow omits anything that would obscure the logic: no error wrapping, no debug code, no retries, no dependency injection boilerplate.

Flow functions are implemented as written — same names, same signatures, same orchestration structure. The only permitted deviation is renaming calls to external resources to fit module structure (e.g. `fetchUser(id)` → `users.Fetch(id)`).

### Writing Flows

Write in the language the feature is implemented in, using an idealized dialect:

- **Pass errors through.** Return errors from helpers directly. Only define a named error type when the error is a distinct domain condition with no underlying cause (e.g. `CartError.Empty` is a state, not a failure).
- **No debug code.** Remove all logger calls, debug statements, and diagnostic output.
- **No dependency injection inside flow functions.** Call functions directly by name. Dependencies are wired above or before the flow is invoked.
- **No infrastructure types.** Use domain nouns, not framework types like request contexts, HTTP writers, etc.
- **Only write bodies that are pure orchestration.** Every line must be a domain condition, a named step call, or a return value. If writing the body would require literals, string construction, or format details, don't write it — just call the function by name.

A flow should be short enough to read in one pass — typically under 20 lines. If it grows longer, split it into named sub-flows.

### Writing Tests

Tests cover domain logic only — the orchestration decisions the flow makes, not the behavior of helpers it calls.

**Test bodies follow the same rule as flow bodies.** Every line should be a domain-language call — setup, invocation, or assertion. If an assertion requires a literal value, a URL, a file path, or any format detail, extract it into a named test helper. A test body that contains raw strings or construction logic has leaked implementation detail.

**Each test covers one domain outcome.** Tests should read as: given this domain state, calling the flow produces this domain result.

### File Format

````markdown
# Checkout Flow

## Purpose
Process a cart purchase and notify the user.

## Flow

```typescript
async function checkout(cart: Cart, user: User) {
    if (cart.isEmpty()) return CartError.Empty

    const order = createOrder(cart, user)

    if (!await chargePayment(order)) {
        cancelOrder(order)
        await sendDeclinedEmail(order, user)
        return
    }

    await sendConfirmationEmail(order, user)
    return order
}
```

### Helpers

- **`createOrder(cart, user)`** — builds an order record from the cart contents and user identity
- **`chargePayment(order)`** — attempts to collect payment for the order; returns whether it succeeded
- **`cancelOrder(order)`** — marks the order as void so it is never fulfilled
- **`sendConfirmationEmail(order, user)`** — notifies the user that their purchase was successful
- **`sendDeclinedEmail(order, user)`** — notifies the user that their payment was not accepted

## Tests

```typescript
test("successful checkout", () => {
    const user = aUser()
    const cart = aCart().withItems(anItem())
    const order = checkout(cart, user)
    expect(order).toMatchOrder(cart, user)
    expect(emailsSent()).toContain(confirmationEmail(order, user))
})

test("payment declined", () => {
    const user = aUser()
    const cart = aCart().withItems(anItem())
    paymentWillDecline()
    checkout(cart, user)
    expect(ordersCreated()).toBeEmpty()
    expect(emailsSent()).toContain(declinedEmail(user))
})

test("empty cart", () => {
    const result = checkout(emptyCart(), aUser())
    expect(result).toBe(CartError.Empty)
})
```

### Helpers

- **`aUser()`** — returns a valid user in a default state suitable for checkout
- **`aCart()`** — returns a cart builder; call `.withItems(...)` to populate it
- **`anItem()`** — returns a purchasable item with a non-zero price
- **`emptyCart()`** — returns a cart with no items
- **`paymentWillDecline()`** — configures the test environment so the next payment attempt fails
- **`ordersCreated()`** — returns the list of orders persisted during the test
- **`emailsSent()`** — returns the list of emails sent during the test
- **`confirmationEmail(order, user)`** — constructs the expected confirmation email value for assertion
- **`declinedEmail(user)`** — constructs the expected declined-payment email value for assertion
````

| Element | Purpose |
|---------|---------|
| `## Purpose` | One sentence describing what the flow accomplishes |
| `## Flow` | The implementation contract — function names, signatures, and orchestration are fixed; only external resource calls may be renamed to fit module structure |
| `## Flow > ### Helpers` | One line per helper called in the flow, describing its domain role |
| `## Tests` | The test contract — test names, bodies, and helper vocabulary the implementation must match |
| `## Tests > ### Helpers` | One line per test helper, describing what domain state it sets up or asserts |

### Where Flows Live

See [Directory Structure](./README.md#directory-structure) for how flows are organized.

### What Flows Are Not

- **Not a spec.** Flows do not define behavioral guarantees. Put those in `/specs`.
- **Not a branching tree.** Flows should be exhaustive but designed to minimize paths. If a flow has many branches, that is a signal to simplify the design, not to add more cases.
