version := `cat internal/version/VERSION`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`
git_commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`

binary := "ralph"
main_path := "./cmd/ralph"
install_path := `go env GOPATH` + "/bin"

ldflags := "-X main.Date=" + build_date

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
    podman build -t "$image" -f Containerfile .

# Push container image to registry
push:
    ./scripts/push-image.sh

# Submit every project in projects/ as a remote workflow
run-projects:
    ./scripts/run-projects.sh
