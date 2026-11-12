# Container Design

The Ralph image runs a remote workflow. It carries Ralph, the tools Ralph calls, and OpenCode. A workflow that needs a language toolchain uses a toolchain image layered on top.

## Contents

The image carries:

* the Ralph binary
* the argo command line
* OpenCode, pinned, with the global configuration
* git and gh, which Ralph calls directly
* a shell and CA certificates

It carries no language toolchain, no build tool, and no browser. It has no Go, Bun, Node, Rust, Python, air, templ, helm, just, Playwright, or Chromium.

The image is the executor, not the development environment. A workflow clones a repository and the agent builds and tests it. That work needs the repository's toolchain, which differs per repository. One image with every toolchain makes every run download and store all of them.

## Base

The runtime stays on `ubuntu:24.04`, pinned by tag.

* It uses glibc, which the OpenCode native binary links against.
* It is the Ubuntu release the Playwright project supports, so a browser image can use Playwright's own install path.
* It is the distribution most project install scripts and third-party apt sources target.
* It carries no browser, Node runtime, or desktop library.

Alpine is rejected because musl breaks the OpenCode binary and most language toolchains. Distroless is rejected because the agent needs a shell, git, and package installation.

The apt step upgrades the base, installs with `--no-install-recommends`, and clears the apt lists in the same layer.

## Stages

The Containerfile has four stages:

1. **Ralph** - a Go image installs `just` and builds the binary with the version and build date.
2. **Argo** - a Go image builds the argo CLI from the release tag with the go-jose override. Argo is the only tool built from source for a patched dependency.
3. **OpenCode** - a Bun image installs the pinned `@opencode/cli` package and keeps only the native binary for this platform. The module cache and the other platform packages stay out of the image.
4. **Runtime** - the Ubuntu base installs its packages and copies the argo CLI, the OpenCode binary, and the Ralph binary. It writes the global OpenCode configuration.

The runtime carries no Go module cache, Bun installation, or npm package tree. The OpenCode binary is about 240 MB and argo is about 100 MB. Keep the image under 650 MB.

## Toolchains

Toolchain images live in `zon/ralph-containers`. Each starts from the Ralph image at a pinned version and adds one toolchain. The repository publishes them to GHCR, and a project selects one with `workflow.image`. See [Config Design](app/ralph-config.md).

The Go image, `ralph-go`, adds:

* the Go toolchain at a pinned version
* air at a pinned version
* templ at a pinned version

air and templ are built from source in a Go stage with their dependencies raised, because their releases still carry a known HIGH finding. An override goes away once a release ships the fixed version.

Browser automation follows the same pattern. A repository that needs browsers selects a toolchain image that installs them. The base image does not carry them.

## Debug

`--debug <branch>` builds Ralph from source in the container with `go run`, so it needs the Go toolchain. The default image does not have it. The debug wrapper checks for `go` and exits with a message that names the toolchain image when it is missing. A developer runs debug with `workflow.image` pointing at `ralph-go`.
