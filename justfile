version := `cat internal/version/VERSION`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`
git_commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`

binary := "ralph"
webhook_binary := "ralph-webhook"
main_path := "./cmd/ralph"
webhook_main_path := "./cmd/ralph-webhook"
install_path := `go env GOPATH` + "/bin"

ldflags := "-X main.Date=" + build_date

# List available recipes
default:
    @just --list

# Build the ralph and ralph-webhook binaries
build:
    @echo "Building {{binary}} v{{version}}..."
    go build -ldflags "{{ldflags}}" -o {{binary}} {{main_path}}
    @echo "Build complete: ./{{binary}}"
    @echo "Building {{webhook_binary}} v{{version}}..."
    go build -ldflags "{{ldflags}}" -o {{webhook_binary}} {{webhook_main_path}}
    @echo "Build complete: ./{{webhook_binary}}"

# Install ralph and ralph-webhook to GOPATH/bin
install:
    @echo "Installing {{binary}} v{{version}} to {{install_path}}..."
    go install -ldflags "{{ldflags}}" {{main_path}}
    @echo "Installation complete: {{install_path}}/{{binary}}"
    @echo "Installing {{webhook_binary}} v{{version}} to {{install_path}}..."
    go install -ldflags "{{ldflags}}" {{webhook_main_path}}
    @echo "Installation complete: {{install_path}}/{{webhook_binary}}"

# Display version information
show-version:
    @echo "Version: {{version}}"
    @echo "Build Date: {{build_date}}"
    @echo "Git Commit: {{git_commit}}"

# Remove built binaries
clean:
    rm -f {{binary}} {{webhook_binary}}

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
