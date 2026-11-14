# Deployment Design

Ralph ships as a binary and as a container image. The binary runs local, worktree, and command line work. The image runs remote workflows.

## Binary

`go install github.com/zon/ralph/cmd/ralph@latest` installs the command. Homebrew installs it from the `zon/tap` tap. `just build` and `just install` build from source. The version comes from `internal/version/VERSION`. A build outside the main branch appends `-dev`, and the linker injects the build date.

## Image

The image is `ghcr.io/zon/ralph`, tagged with the version. `workflow.image` in the [config](app/ralph-config.md) overrides the repository and tag, so a workflow can pin a different image. Toolchain images under `containers/` layer a language toolchain on the Ralph image and share its version. See [Container Design](container.md).

The Containerfile has four stages:

* A Go builder compiles the binary with the version and build date.
* A Go stage builds the Argo CLI from source at a pinned version. The latest release still ships a dependency with a known HIGH vulnerability, so the stage raises that dependency and keeps the Go module cache out of the runtime image. An override goes away once a release of the tool ships the fixed version.
* A Bun stage installs the pinned OpenCode package and keeps only the native binary for this platform.
* An Ubuntu runtime provides git and the GitHub CLI, then adds the Argo CLI, OpenCode, and the Ralph binary. It carries no language toolchain and no browser.

The image also carries a global OpenCode configuration that lets the agent act without a client to answer permission prompts. The container boundary is the sandbox. Web search selects its provider automatically, because the first search otherwise asks the user to choose one, a prompt an unattended run cannot answer. A failed search is a tool error, not a stalled run, and the agent still fetches pages with webfetch and the shell. The image disables OpenCode updates, so the pinned version is the version every run uses.

`just container-build` builds locally and `just push` pushes the image only when the scan passes. `just container-build-go`, `just container-test-go`, and `just push-go` do the same for the Go toolchain image.

`scripts/scan.sh` owns scanning. It runs `govulncheck` against the Go dependencies reachable from this module, then Trivy against the working tree or a container image, and stops as soon as either reports a problem. `just scan` runs the script with no arguments to scan the working tree. Callers that scan an image run the script instead of calling Trivy directly.

## Delivery

GitHub Actions builds and delivers the project:

* A push to `main` or `dev` builds the image, scans it with Trivy, and pushes it only when the scan passes. `main` gets the version tag and `latest`. `dev` gets the version plus `-dev` and `dev`.
* A push to `main` or `dev` packages every chart under `charts/` and pushes it to `oci://ghcr.io/zon/charts`. `main` uses the chart version and `dev` appends `-dev`, for example `0.3.1-dev`.
* A pull request builds the image, scans it with Trivy, and keeps the reports as artifacts. A high or critical finding fails the scan.
* A version change creates a GitHub release and tags it. On `main` it also points the Homebrew tap at the new source archive. A `dev` release is a prerelease and leaves the tap alone.

The image and the release share the version in `internal/version/VERSION`.

## Credentials

A deployment supplies credentials as Kubernetes Secrets:

* `github-credentials` holds either App credentials or a token. A token needs the `workflow` scope to push changes to workflow files.
* `opencode-credentials` holds `auth.json`.

`ralph setup` writes both. The workflow mounts them at `/secrets/github` and `/secrets/opencode`. See [Setup Design](setup.md) and [Workflow Container Design](workflow.md).
