# Outline: Major Features

Analyze the repo and produce `outlines/major-features.md` describing the project's major features.

## What is a Major Feature

Read [docs/standards/major-features.md](../standards/major-features.md) to understand what qualifies as a major feature before proceeding.

## How to Analyze the Repo

1. Read the codebase to understand what the software does at a high level.
2. Group related behavior by the real-world concern it addresses. Each group is a candidate major feature.
3. Discard infrastructure concerns (routing, database access, configuration loading) — these are not major features.
4. Name each feature as a noun phrase a non-technical stakeholder would recognize.

## Output Format

Produce `outlines/major-features.md` at the root of the repo. The file must have one section per major feature:

```markdown
## <Feature Name>

<One or two sentence description of what the feature does and why it exists.>

- <Business logic bullet 1>
- <Business logic bullet 2>
- <Business logic bullet 3>
```

Each bullet point must describe a specific business rule or process step — not a technical implementation detail. Write bullets from the perspective of what the software enforces or orchestrates, not how it does it internally.

**Good bullet examples:**
- Only the message author or a channel admin may delete a message
- A payment is rejected if the account balance is insufficient
- A review is required from at least one code owner before a PR can merge

**Bad bullet examples:**
- Calls `DeleteMessage` in the repository layer
- Uses a SQL transaction to update the balance
- Sends an HTTP request to the GitHub API
