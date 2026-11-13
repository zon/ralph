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

The Containerfile has six stages:

1. **Ralph** - a Go image installs `just` and builds the binary with the version and build date.
2. **Git** - an Ubuntu image installs git, whose package depends on the full Perl runtime. The runtime copies only the compiled git commands and templates.
3. **Argo** - a Go image builds the argo CLI from the release tag with the go-jose and x/net overrides.
4. **GitHub CLI** - a Go image builds the GitHub CLI from the release tag with the x/net override. Argo and the GitHub CLI are built from source for a patched dependency until a release ships the fix.
5. **OpenCode** - a Bun image installs the pinned `@opencode/cli` package and keeps only the native binary for this platform. The module cache and the other platform packages stay out of the image.
6. **Runtime** - the Ubuntu base installs its packages and copies git, the argo CLI, the GitHub CLI, the OpenCode binary, and the Ralph binary. It writes the global OpenCode configuration.

The runtime carries no Go module cache, Bun installation, or npm package tree. The OpenCode binary is about 240 MB and argo is about 100 MB. Keep the image under 650 MB.

## Toolchains

Container images live in this repository under `containers/`. The Ralph image is in `containers/ralph`, and each toolchain image has its own directory beside it. Each toolchain image starts from the Ralph image at the same version and adds one toolchain. The container build workflow pushes each one to GHCR right after the Ralph image, with the same version and alias tags, and a project selects one with `workflow.image`. See [Config Design](app/ralph-config.md).

The base is the `RALPH_IMAGE` build argument, which defaults to `ghcr.io/zon/ralph` at `RALPH_VERSION`. A pull request builds the toolchain image on the Ralph image it just built, so a version that is not published yet still builds and scans.

The Go image, `ralph-go`, is built from `containers/ralph-go/Containerfile` and adds:

* the Go toolchain at a pinned version
* air at a pinned version
* templ at a pinned version
* govulncheck at a pinned version

air and templ are built from source in a Go stage with their dependencies raised, because their releases still carry a known HIGH finding. An override goes away once a release ships the fixed version. govulncheck is built from source in the same stage, next to them, so it is available to `scripts/scan.sh` inside the container. It reads the Go vulnerability database from `vuln.go.dev`, so a container that runs it needs network access.

Browser automation follows the same pattern. A repository that needs browsers selects a toolchain image that installs them. The base image does not carry them.

## Debug

`--debug <branch>` builds Ralph from source in the container with `go run`, so it needs the Go toolchain. The default image does not have it. The debug wrapper checks for `go` and exits with a message that names the toolchain image when it is missing. A developer runs debug with `workflow.image` pointing at `ralph-go`.
