base_version := `cat internal/version/VERSION`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`
git_commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
branch := `git branch --show-current 2>/dev/null || echo "unknown"`

# Append -dev when built outside the main branch
version := base_version + if branch == "main" { "" } else { "-dev" }

binary := "ralph"
main_path := "./cmd/ralph"
install_path := `go env GOPATH` + "/bin"

ldflags := "-X main.Date=" + build_date + " -X github.com/zon/ralph/internal/version.BuildVersion=" + version

# List available recipes
default:
    @just --list

# Build the ralph binary
build:
    @echo "Building {{binary}} v{{version}}..."
    go build -ldflags "{{ldflags}}" -o {{binary}} {{main_path}}
    @echo "Build complete: ./{{binary}}"

# Install ralph to GOPATH/bin
install:
    @echo "Installing {{binary}} v{{version}} to {{install_path}}..."
    go install -ldflags "{{ldflags}}" {{main_path}}
    @echo "Installation complete: {{install_path}}/{{binary}}"

# Display version information
show-version:
    @echo "Version: {{version}}"
    @echo "Build Date: {{build_date}}"
    @echo "Git Commit: {{git_commit}}"

# Remove built binaries
clean:
    rm -f {{binary}}

# Run tests
test:
    go test -v ./...

# Build container image
container-build:
    #!/usr/bin/env bash
    repository="ghcr.io/zon/ralph"
    image="$repository:{{version}}"
    echo "Building container $image..."
    podman build -t "$image" -f containers/ralph/Containerfile .

# Build the Go toolchain image on the local Ralph image
container-build-go: container-build
    podman build \
      --build-arg RALPH_IMAGE=ghcr.io/zon/ralph:{{version}} \
      --build-arg RALPH_VERSION={{version}} \
      -t ghcr.io/zon/ralph-go:{{version}} \
      -f containers/ralph-go/Containerfile containers/ralph-go

# Confirm the Go toolchain image carries the Go toolchain, air, templ, and govulncheck
container-test-go: container-build-go
    podman run --rm --entrypoint /bin/sh ghcr.io/zon/ralph-go:{{version}} -lc \
      'go version && command -v air && templ version && govulncheck -version'

# Scan the working tree for vulnerabilities
scan:
    ./scripts/scan.sh

# Scan then push the Go toolchain image to the registry
push-go:
    #!/usr/bin/env bash
    set -euo pipefail
    ./scripts/scan.sh image "ghcr.io/zon/ralph-go:{{version}}"
    podman push "ghcr.io/zon/ralph-go:{{version}}"

# Push container image to registry
push:
    ./scripts/push-image.sh

# Release dev to main
release:
    #!/usr/bin/env bash
    set -euo pipefail

    branch="$(git branch --show-current)"
    if [ "$branch" != "dev" ]; then
        echo "error: release must run on the dev branch (currently on ${branch})" >&2
        exit 1
    fi

    git fetch origin dev
    ahead="$(git rev-list --count origin/dev..dev)"
    behind="$(git rev-list --count dev..origin/dev)"
    if [ "$behind" -gt 0 ]; then
        echo "error: dev is ${behind} commit(s) behind origin/dev; pull before releasing" >&2
        echo "run: git pull origin dev" >&2
        exit 1
    fi
    if [ "$ahead" -gt 0 ]; then
        echo "dev is ${ahead} commit(s) ahead of origin/dev; pushing to origin first"
        git push origin dev
    fi

    echo "Pushing dev to main..."
    git push origin dev:main

# Submit every project in projects/ as a remote workflow
run-projects:
    ./scripts/run-projects.sh
