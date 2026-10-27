---
name: update-dependency
description: Finds an outdated dependency, reads its release notes, and applies one update with verification. Use when the user asks to update, upgrade, or bump a dependency, or to check for outdated packages.
---

# Update Dependency

Apply one dependency update per run, so the change stays small enough to review.

## Steps

1. **List direct dependencies.** Read every manifest and lockfile in this repository, such as `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, and `requirements.txt`. Omit transitive dependencies that no manifest declares.

2. **Order the dependencies.** Put small standalone dependencies first and large frameworks and platforms last.

3. **Find one update.** Check each dependency from top to bottom with the package manager, such as `go list -m -u all`, `npm outdated`, or `cargo update --dry-run`. Confirm the version on the package registry when the command cannot. Skip prereleases. Stop at the first dependency that has a newer stable version.

4. **Read the release notes.** Read the changelog for every skipped version and note breaking changes. A breaking change, a major version, or a changed component boundary, public API, or data model is significant. Otherwise the change is small.

5. **Apply the update.**
   - Update the manifest and lockfile with the package manager, not by hand.
   - For a small change, fix the code the update breaks.
   - For a significant change, write a spec and project as described in [docs/zpecs/specs.md](docs/zpecs/specs.md) and [docs/zpecs/project.md](docs/zpecs/project.md), and a design under `docs/design`, before coding.
   - Follow the repository's rules for changelogs and version bumps.

6. **Verify.** Run the build and tests, and fix what the update breaks before reporting.

7. **Report** the dependency, the old and new versions, whether the change was small or significant, and the verification result. If no update exists, report that and stop.
