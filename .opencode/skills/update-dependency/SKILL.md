---
name: update-dependency
description: Finds an outdated direct dependency, reads its release notes, and applies one update with verification. Use when the user asks to update, upgrade, or bump a dependency, or to check for outdated packages.
---

# Update Dependency

Apply one direct dependency update per run, so the change stays small enough to review.

## Definitions

- **Direct dependency** — one the project declares for itself. A transitive dependency comes through another dependency, so the project cannot update it on its own.
- **Allowed range** — the versions a manifest constraint accepts.
- **Resolved version** — the version a lock file or pin selects. A lock file is part of the dependency, not a build artifact.
- **Newest stable** — the highest version the registry publishes that is not a prerelease.
- **Outdated** — a dependency whose resolved version is below the newest stable, whether or not the allowed range includes it.
- **Unpinned** — a dependency with no version, or a floating tag such as `latest`. It cannot be outdated.
- **Held back** — a dependency the project keeps at its current version on purpose, as [docs/zpecs/dependencies.md](docs/zpecs/dependencies.md) describes.

## Steps

1. **List direct dependencies.** Read every dependency manifest in the repository and keep the dependencies the project declares for itself. Leave out transitive dependencies, including entries a manifest marks as indirect.

   Version pins outside a manifest are direct dependencies too. Read container files, CI workflows, and install or infrastructure scripts. A base image tag, a tool version, or a version variable the project sets is a pin it owns. So is a runtime or managed-service version the project selects, such as a language runtime, a cluster or database engine version, or a dated API schema version.

   Read every lock file. One constraint can resolve to different versions in different lock files, and each lock file is a place the pin repeats.

2. **Order the dependencies.** Put small standalone dependencies first and large frameworks and platforms last.

3. **Find one update.** For each dependency, compare the resolved version with the newest stable in its registry, release page, or image tags. A package manager's outdated command usually reports only inside the allowed range, so use it to find resolved versions, not to decide what is outdated. When the newest stable sits outside the allowed range, widening the range is part of the update. A range that excludes a major version is not a decision to stay on the old one. Only a held back entry, or a comment that names the reason, is. Skip a held back dependency and report it.

   Skip prereleases. Do not pin an unpinned dependency unasked. Stop at the first outdated direct dependency. When none is outdated, report that with the resolved version and newest stable of each dependency checked, and stop, even if transitive dependencies are outdated.

4. **Read the release notes.** Read the changelog for every skipped version. Note each breaking change and check whether the repository uses what it breaks. A major version is a reason to read closely, not a verdict on its own.

5. **Check compatibility.** Find what the new version works with, such as runtimes, platforms, and the dependencies that run beside it or build on it. Release notes and documentation often list them. Compare each with the version the repository uses. When one does not match, skip this dependency, go back to step 3 for the next outdated one, and report the skipped dependency and the reason. When the mismatch will last, add the dependency to the project's held back list, with why and when to lift the hold.

6. **Apply the update.**
   - Update the manifest and lockfile with the package manager, not by hand.
   - Update a pin outside a manifest in the file that sets it, and in every place that repeats it. When copies of one pin already disagree, align every copy to the new version.
   - Fix the code the update breaks.
   - When that fix changes a component boundary, public API, or data model, the change is significant. Write a spec and project as described in [docs/zpecs/specs.md](docs/zpecs/specs.md) and [docs/zpecs/project.md](docs/zpecs/project.md), and a design under `docs/design`, before coding it.
   - Otherwise the change is small. Add no design, spec, or project. An update that passes verification without a code change is small, even across a major version.
   - Follow the repository's rules for changelogs and version bumps.

7. **Verify.** Run the build and tests, and fix what the update breaks before reporting. When the update changes a container image or a tool in it, build the image before reporting.

8. **Report** the dependency, the old and new versions, whether the change was small or significant, and the verification result. Also list every unpinned dependency, every pin whose copies disagreed, every dependency skipped as incompatible, and every held back dependency.
