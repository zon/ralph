# Deployment Design

Ralph ships as a binary and as a container image. The binary runs local, worktree, and command line work. The image runs remote workflows.

## Binary

`go install github.com/zon/ralph/cmd/ralph@latest` installs the command. Homebrew installs it from the `zon/tap` tap. `just build` and `just install` build from source. The version comes from `internal/version/VERSION`. A build outside the main branch appends `-dev`, and the linker injects the build date.

## Image

The image is `ghcr.io/zon/ralph`, tagged with the version. `workflow.image` in the [config](app/ralph-config.md) overrides the repository and tag, so a workflow can pin a different image.

The Containerfile has two stages:

* A Go builder compiles the binary with the version and build date.
* A Playwright runtime provides the browsers, then adds git tooling, the GitHub CLI, Go, Bun, OpenCode, the Argo CLI, Helm, and the Ralph binary.

The image also carries a global OpenCode configuration that lets the agent act without a client to answer permission prompts. The container boundary is the sandbox. The image disables OpenCode updates, so the pinned version is the version every run uses.

`just container-build` builds locally and `just push` pushes the image.

## Delivery

GitHub Actions builds and delivers the project:

* A push to `main` or `dev` builds the image and pushes it. `main` gets the version tag and `latest`. `dev` gets the version plus `-dev` and `dev`.
* A pull request builds the image, scans it with Trivy, and keeps the reports as artifacts. A critical finding fails the scan.
* A version change creates a GitHub release and tags it. On `main` it also points the Homebrew tap at the new source archive. A `dev` release is a prerelease and leaves the tap alone.

The image and the release share the version in `internal/version/VERSION`.

## Credentials

A deployment supplies credentials as Kubernetes Secrets:

* `github-credentials` holds either App credentials or a token. A token needs the `workflow` scope to push changes to workflow files.
* `opencode-credentials` holds `auth.json`.

`ralph setup` writes both. The workflow mounts them at `/secrets/github` and `/secrets/opencode`. See [Setup Design](setup.md) and [Workflow Container Design](workflow.md).
