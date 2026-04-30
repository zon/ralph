## Flows

Flows are loose templates for implementation — idealized code that defines the high-level domain abstraction the real code should be close to.

### What a Flow Is

A flow is a function (or small set of functions) written in the actual language of the feature, capturing the essential shape of a domain process. It omits anything that would obscure the logic: no error wrapping, no logging, no retries, no dependency injection boilerplate.

The objective is to produce implementation code with good high-level domain abstraction. A well-written flow should read almost like the real code, with the real code adding infrastructure around the same domain structure.

### Example

A real implementation of a checkout process involves payment gateway clients, transaction rollback, email service calls, and extensive error handling. The flow covers all paths but strips away everything that isn't domain logic:

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

This is the domain story — exhaustive, but with as few branches as the design allows. The real implementation adds infrastructure around this same structure.

### Writing Flows

Write in the language the feature is implemented in, using an idealized dialect:

- **Errors as domain conditions.** Include error returns that represent meaningful domain outcomes (e.g. `CartError.Empty`, payment declined). Omit low-level errors like I/O failures or network timeouts.
- **No logging.** Remove all logger calls.
- **No dependency injection.** Call functions directly by name.
- **No infrastructure types.** Use domain nouns, not framework types like request contexts, HTTP writers, etc.
- **Inline trivial steps.** If a helper does one obvious thing, fold it into the caller.
- **Only write bodies that are pure orchestration.** A function body belongs in the flow only if every line is a domain condition, a named step call, or a return value. If writing the body would require any literals, string construction, path manipulation, or format details — even via helper calls — don't write it. Just call the function by name from wherever it's needed.
- **Name things clearly.** Function and variable names should read as domain language, not implementation jargon.

A flow should be short enough to read in one pass — typically under 20 lines. If it grows longer, split it into named sub-flows.

### Where Flows Live

See [Directory Structure](./README.md#directory-structure) for how flows are organized.

### Flow File Format

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

**Key elements:**

| Element | Purpose |
|---------|---------|
| `## Purpose` | One sentence describing what the flow accomplishes |
| `## Flow` | The idealized code — exhaustive but designed to minimize branches |
| `## Flow > ### Helpers` | One line per helper called in the flow, describing its domain role |
| `## Tests` | High-level test flows abstracting data setup, logic calls, and assertions |
| `## Tests > ### Helpers` | One line per test helper, describing what domain state it sets up or asserts |

### Writing Tests

Tests cover domain logic only — the orchestration decisions the flow makes, not the behavior of helpers it calls. If a domain test needs to verify something a helper does (e.g. link rewriting produces the right output), that assertion belongs in a named test helper, not inline in the domain test.

**Don't test helpers directly.** A test that calls a helper like `formatOutput` or `fetchRecord` directly is testing helper behavior, not domain logic. Write domain tests that exercise the full flow and use test helpers to set up or assert helper-level detail.

**Test bodies follow the same rule as flow bodies.** Every line in a test should be a domain-language call — setup, invocation, or assertion. If an assertion requires a literal value, a URL pattern, a file path, or any other format detail, extract it into a named test helper. A test body that contains raw strings or construction logic has leaked implementation detail.

**Each test covers one domain outcome.** Tests should read as: given this domain state, calling the flow produces this domain result.

### What Flows Are Not

- **Not a spec.** Flows do not define behavioral guarantees or test scenarios. Put contracts in `/specs`.
- **Not exact implementation.** Flows are loose templates — the real code adds infrastructure around the same domain structure, but the high-level shape should remain close.
- **Not a branching tree.** Flows should be exhaustive but designed to minimize paths. If a flow has many branches, that is a signal to simplify the design, not to add more cases.