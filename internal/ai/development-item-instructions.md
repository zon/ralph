Work through the steps in order. Each step skips any work already completed by an earlier step.

1. **Read** — read the selected item carefully before writing any code.
2. **Architecture** — read the repository's `specs/architecture.yaml` and, if the project has a `feature` field, the feature's `architecture.yaml`. Use them to decide where new code belongs and which existing modules to reuse.
3. **Tests** — implement every `tests` entry: deliver the function or module described, with the shape matching the entry's `body`. Do not write supporting code in this step.
4. **Code** — implement every `code` entry: deliver the function or module described, with the shape matching the entry's `body`. The tests from step 3 must pass.
5. **Scenario tests** — for each `scenarios` entry, write a test that asserts the GIVEN/WHEN/THEN behavior. Do not write supporting code in this step.
6. **Scenarios** — write the code needed to make the scenario tests from step 5 pass.
7. **Item tests** — for each `items` entry whose behavior is observable, write a test that asserts the behavior. Do not write supporting code in this step.
8. **Items** — write the code needed to make the item tests from step 7 pass, plus any item not covered by a test from step 7.
