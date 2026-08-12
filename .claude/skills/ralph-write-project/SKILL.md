---
name: ralph-write-project
description: Creates and validates a ralph project file defining work for the ralph agent to execute
---

# Write Project

Create a well-formed project file based on the user's description of the work to be done.

A project is any YAML or JSON file containing an array of work items. Ralph imposes no schema on the file — it resolves the array with a jq **item query** and treats everything else in the file as opaque context passed through to the agent. Each element of the array is an **item**: one unit of work, one iteration.

An item is identified by its **index**, its 0-based position in the resolved array. When the item is a mapping with a scalar `slug`, `id`, or `name` field — checked in that order — that value is the item's **key**, a label used in commit messages, logs, and picker output. No completion field belongs in the file: completion is recorded in the branch's commit messages, never written back to the project file. Top-level `slug` and `title` are optional and fall back to the file name.

## Steps

1. **Understand the work.** If the user's request is vague, ask clarifying questions:
   - What feature or change does this project cover?
   - Does it target a documented feature directory under `specs/features/`?
   - Does the work require a version bump?

2. **Locate the feature directory** if the project targets a documented feature. Feature directories live under `specs/features/<component>/<feature>/` and may contain any of `spec.md`, `orchestration.md`, and `architecture.yaml` — all optional.

3. **Read the project format docs** to refresh your understanding:
   - [docs/formats/project.md](docs/formats/project.md)

4. **Read the coding and testing standards** so items are consistent with how this codebase is written and tested:
   - [docs/code.md](docs/code.md)
   - [docs/testing.md](docs/testing.md)

5. **Check the module category** for every module the items will touch. Read [specs/architecture.yaml](specs/architecture.yaml). If the project targets a feature and `<feature-dir>/architecture.yaml` is present, read that too — it describes modules introduced or changed by the feature. Look up the `category` field for each affected module path. The category's `signatures` and `orchestration` flag (defined in [docs/formats/architecture.md](docs/formats/architecture.md)) determine what form the code and tests must take. Apply those constraints when writing `items`, `code`, and `tests` for each item.

6. **Draft orchestration-based items.** If `<feature-dir>/orchestration.md` is present, read it and create an item for each implementation shape it defines. Source `code` and `tests` entries exclusively from it — never invent shapes.

7. **Draft scenario-based items.** If `<feature-dir>/spec.md` is present, read it and add its scenarios to any matching items from step 6. If a scenario doesn't correspond to an orchestration item, create a new item for it with `scenarios` only.

8. **Draft remaining items** for any work not covered by the orchestration or spec — additional constraints, edge cases, operational requirements, and the version bump if needed.

9. **Write the file** at `./projects/<slug>.yaml`, following the format and guidelines in [docs/formats/project.md](docs/formats/project.md). Write the items in the conventional shape — a top-level mapping whose `requirements` list holds the item array, paired with `items: .requirements` in config — and give each item a key:

   ```yaml
   slug: project-identifier        # branch name (ralph/<slug>)
   title: Brief description        # PR title

   feature: specs/features/<component>/<feature>   # optional: link to feature directory

   requirements:
     - slug: requirement-identifier
       description: What should happen
       items:
         - Specific behavioral outcome the agent must achieve
       scenarios:
         - title: Scenario title
           items:
             - GIVEN ...
             - WHEN ...
             - THEN ...
       code:
         - name: ExampleFunc
           description: optional summary of what this function does
           module: path/to/module
           body: |
             func ExampleFunc() {
               // target implementation shape
             }
       tests:
         - name: TestExampleFunc
           description: verifies ExampleFunc handles the happy path
           module: path/to/module
           body: |
             func TestExampleFunc(t *testing.T) {
               // assertions
             }
   ```

   Each item's `slug`, `id`, or `name` field — checked in that order — is its key, so commit trailers such as `Ralph item 0 (csv-serializer) completed` read well. No completion field belongs in the file: completion is recorded in commit messages, never written into the project file.

10. **Validate** by running `ralph validate ./projects/<slug>.yaml`. This checks the file parses and the item query resolves to at least one non-empty item. Empty entries — null, `false`, `0`, blank strings, `{}`, `[]` — are dropped during resolution, so never leave a placeholder entry in the list expecting ralph to work it.

11. **Report** the file path and a one-line summary of what the project covers.
