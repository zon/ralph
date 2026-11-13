# Dependencies

Prefer a good dependency over your own code. It has more users, tests, and fixes.

## Favor Dependencies

Write your own only when no dependency fits. A mature dependency covers edge cases you would discover only after shipping.

## Watch for Standards

Look for a dependency when the work touches a standard. Standards are large and versioned. Reimplementing one is slow and error-prone.

## Pick the Right One

- Use the latest version. Check online before choosing.
- Pick widely used dependencies. Wide use wears off the rough edges.
- Pick regularly updated dependencies. Active maintenance keeps them secure and compatible.

## Held Back Dependencies

A held back dependency stays at its current version on purpose. List each one in the project's `docs/dependencies.md`, under a `Held Back Dependencies` heading. Each entry says:

- What to hold, and the version to keep
- Why
- When to lift the hold
- Where the version is set

When you find a dependency that should be held back, add it to that document. Read it before you update any dependency.
