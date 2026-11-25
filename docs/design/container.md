# Container Design

The Ralph image runs a remote workflow. It carries Ralph, the tools Ralph calls, and OpenCode. A workflow that needs a language toolchain uses a toolchain image layered on top.

## Contents

The image carries:

* the Ralph binary
* the argo command line
* OpenCode, pinned, with the global configuration
* git and gh, which Ralph calls directly
* a shell and CA certificates

It carries no language toolchain, no build tool, and no browser. It has no Go, Bun, Node, Rust, Python, air, templ, helm, or just.

The image is the executor, not the development environment. A workflow clones a repository and the agent builds and tests it. That work needs the repository's toolchain, which differs per repository. One image with every toolchain makes every run download and store all of them.

## Base

The runtime stays on `ubuntu:26.10`, pinned by tag.

* It uses glibc, which the OpenCode native binary links against.
* It is the distribution most project install scripts and third-party apt sources target.
* It carries no browser, Node runtime, or desktop library.

Alpine is rejected because musl breaks the OpenCode binary and most language toolchains. Distroless is rejected because the agent needs a shell, git, and package installation.

The apt step upgrades the base, installs with `--no-install-recommends`, and clears the apt lists in the same layer.

## Stages

The Containerfile has five stages:

1. **Ralph** - a Go image installs `just` and builds the binary with the version and build date.
2. **Git** - an Ubuntu image installs git, whose package depends on the full Perl runtime. The runtime copies only the compiled git commands and templates.
3. **Argo** - a Go image builds the argo CLI from the release tag with the go-jose and x/net overrides.
4. **OpenCode** - a Bun image installs the pinned `@opencode/cli` package and keeps only the native binary for this platform. The module cache and the other platform packages stay out of the image.
5. **Runtime** - the Ubuntu base installs its packages, including gh, and copies git, the argo CLI, the OpenCode binary, and the Ralph binary. It removes the base image's Pebble service manager, which the container never starts, and writes the global OpenCode configuration.

The runtime carries no Go module cache, Bun installation, or npm package tree. The OpenCode binary is about 240 MB and argo is about 100 MB. Keep the image under 650 MB.

## Review Image

The review image, `ralph-review`, adds the tools a workflow needs to work with a code review report and the spec definitions. It starts from the Ralph image at the same version.

Its Containerfile has three stages:

1. **zpecs** - a Go image builds the zpecs CLI from the specs repository at a pinned commit. The repository publishes no release tags, so the commit is the pin, and the build injects the version from its `VERSION` file.
2. **Open Code Review** - an Ubuntu image downloads the pinned `opencodereview-linux-amd64` release asset and verifies its published SHA-256 checksum. The asset is a static binary, so the image needs no JavaScript runtime.
3. **Runtime** - the Ralph image adds zpecs and `ocr`.

The Go toolchain image builds on the review image, so it carries both commands.

## OpenCode Configuration

The runtime writes a global OpenCode configuration so every run inherits it. The configuration allows every tool action, because the container is the sandbox and a run has no client to answer a rule that asks. It selects the web search provider automatically, because the first search otherwise asks the user to choose one, another prompt an unattended run cannot answer. A failed search is a tool error, not a stalled run, and the agent can still fetch pages with webfetch and the shell. It disables updates, so the pinned OpenCode version is the version every run uses.

## Toolchains

Container images live in this repository under `containers/`. The Ralph image is in `containers/ralph`, the review image in `containers/ralph-review`, and each toolchain image has its own directory beside it. Each toolchain image starts from an image at the same version and adds one toolchain. The container build workflow pushes each one to GHCR after the image it layers on, with the same version and alias tags, and a project selects one with `workflow.image`. See [Config Design](app/ralph-config.md).

The base is the `RALPH_IMAGE` build argument, which defaults to the image the Containerfile layers on at `RALPH_VERSION`. The review image layers on `ghcr.io/zon/ralph`, and the Go image layers on `ghcr.io/zon/ralph-review`. The container build workflow builds each image after it pushes the image it layers on.

The Go image, `ralph-go`, is built from `containers/ralph-go/Containerfile` and adds:

* the Go toolchain at a pinned version
* air at a pinned version
* templ at a pinned version
* govulncheck at a pinned version

air and templ are built from source in a Go stage with their dependencies raised, because their releases still carry a known HIGH finding. An override goes away once a release ships the fixed version. govulncheck is built from source in the same stage, next to them, so it is available to `scripts/scan.sh` inside the container. It reads the Go vulnerability database from `vuln.go.dev`, so a container that runs it needs network access.

Browser automation follows the same pattern. A repository that needs browsers selects a toolchain image that installs them. The base image does not carry them.

## Debug

`--debug <branch>` builds Ralph from source in the container with `go run`, so it needs the Go toolchain. The default image does not have it. The debug wrapper checks for `go` and exits with a message that names the toolchain image when it is missing. A developer runs debug with `workflow.image` pointing at `ralph-go`.
