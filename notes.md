# Notes

## Feature architecture files must be moved, not deleted

`specs/features/<area>/<feature>/architecture.yaml` is a staging file for modules planned but not yet coded. After implementing a module, ralph must **move** its entry from the feature architecture file into the root `specs/architecture.yaml` — then delete the feature file once all its modules have been promoted.

What went wrong: ralph deleted a feature architecture file without first moving its module entries into the root `specs/architecture.yaml`. Instructions should make explicit that deleting a feature architecture file is only valid after every module it contains has been added to the root file.

## Architecture module descriptions

Instructions should say descriptions must be a single short sentence stating the module's purpose and role — no method names, route lists, interface names, or error types. Details like these churn every time the module grows. A good description should survive multiple features being added without needing an edit.
