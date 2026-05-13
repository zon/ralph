# Flow Format

The flow format defines implementation contracts using idealized code. A flow document specifies function names, orchestration structure, and helper vocabulary that the real implementation must preserve. Flows omit anything that would obscure the logic: no error wrapping, no debug code, no retries, no dependency injection boilerplate.

Flow functions are implemented as written — same names, same signatures, same orchestration structure.

## Structure

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

## File Format

````markdown
# Checkout Flow

## Purpose
Process a cart purchase and notify the user.

## Flow

**Module:** `checkout`

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

- **`createOrder(cart, user)`** [`orders`] — builds an order record from the cart contents and user identity
- **`chargePayment(order)`** [`payments`] — attempts to collect payment for the order; returns whether it succeeded
- **`cancelOrder(order)`** [`orders`] — marks the order as void so it is never fulfilled
- **`sendConfirmationEmail(order, user)`** [`email`] — notifies the user that their purchase was successful
- **`sendDeclinedEmail(order, user)`** [`email`] — notifies the user that their payment was not accepted

## Tests

**Module:** `checkout.test`

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

- **`aUser()`** [`users.fixtures`] — returns a valid user in a default state suitable for checkout
- **`aCart()`** [`cart.fixtures`] — returns a cart builder; call `.withItems(...)` to populate it
- **`anItem()`** [`cart.fixtures`] — returns a purchasable item with a non-zero price
- **`emptyCart()`** [`cart.fixtures`] — returns a cart with no items
- **`paymentWillDecline()`** [`payments.fixtures`] — configures the test environment so the next payment attempt fails
- **`ordersCreated()`** [`orders.fixtures`] — returns the list of orders persisted during the test
- **`emailsSent()`** [`email.fixtures`] — returns the list of emails sent during the test
- **`confirmationEmail(order, user)`** [`email.fixtures`] — constructs the expected confirmation email value for assertion
- **`declinedEmail(user)`** [`email.fixtures`] — constructs the expected declined-payment email value for assertion
````

| Element | Purpose |
|---------|---------|
| `## Purpose` | One sentence describing what the flow accomplishes |
| `## Flow` | The implementation contract — function names, signatures, and orchestration are fixed |
| `## Flow > **Module**` | The module where the flow function is implemented |
| `## Flow > ### Helpers` | One line per helper called in the flow, with its module in brackets and a description of its domain role |
| `## Tests` | The test contract — test names, bodies, and helper vocabulary the implementation must match |
| `## Tests > **Module**` | The module where the tests are implemented |
| `## Tests > ### Helpers` | One line per test helper, with its module in brackets and a description of what domain state it sets up or asserts |

## Module Structure

The flow function lives in an [orchestration module](../glossary.md#orchestration-module). Each helper lives in an [implementation module](../glossary.md#implementation-module).

Every flow document must declare its module assignments. The `**Module:**` line under `## Flow` names the orchestration module; the `**Module:**` line under `## Tests` names the test module. These are implementation contracts — the code must match.

Tests live in the same module they test, or in a companion module with the same name. A test helper that asserts on payment behavior belongs near the module it tests, not in a generic test module.

## File Location

See [Directory Structure](./README.md#directory-structure) for where flow files are located.

## What Flows Are Not

- **Not a spec.** Flows do not define behavioral guarantees. Put those in `/specs`.
- **Not a branching tree.** Flows should be exhaustive but designed to minimize paths. If a flow has many branches, that is a signal to simplify the design, not to add more cases.
