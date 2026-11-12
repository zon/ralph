# Build stage - compile ralph
FROM docker.io/library/golang:1.27-bookworm AS builder

WORKDIR /build

RUN curl -fsSL https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and version files
COPY . .

# Build ralph binary using justfile (includes version injection)
RUN just build

# Go tools stage - build the Go command line tools from source. The latest
# release of each still ships a dependency with a known HIGH vulnerability
# (air: golang.org/x/text, templ: golang.org/x/net, argo: go-jose, helm: the Go
# standard library), and a released binary cannot be patched. Building from the
# release tag lets us raise that dependency, or build with a Go that carries the
# fix, and keeps the Go module cache out of the runtime image. Drop an override
# once a release of that tool ships the fixed version. The Go version here must
# stay at or above the fix for helm's standard library finding (1.27.3).
FROM docker.io/library/golang:1.27.4-bookworm AS go-tools

ENV CGO_ENABLED=0
RUN mkdir /out

# air for live reload, built with a fixed golang.org/x/text
ENV AIR_VERSION=v1.67.4
ENV AIR_TEXT_VERSION=v0.44.0
RUN mkdir /tmp/air && cd /tmp/air \
    && go mod init tools \
    && GOFLAGS=-mod=mod go get github.com/air-verse/air@${AIR_VERSION} golang.org/x/text@${AIR_TEXT_VERSION} \
    && GOFLAGS=-mod=mod go build -trimpath -ldflags "-s -w" -o /out/air github.com/air-verse/air

# templ for templates, built with a fixed golang.org/x/net
ENV TEMPL_VERSION=v0.3.1118
ENV TEMPL_NET_VERSION=v0.62.0
RUN mkdir /tmp/templ && cd /tmp/templ \
    && go mod init tools \
    && GOFLAGS=-mod=mod go get github.com/a-h/templ/cmd/templ@${TEMPL_VERSION} golang.org/x/net@${TEMPL_NET_VERSION} \
    && GOFLAGS=-mod=mod go build -trimpath -ldflags "-s -w" -o /out/templ github.com/a-h/templ/cmd/templ

# Argo Workflows CLI, built with a fixed go-jose. The CLI embeds the UI
# directory, which only the server uses; like upstream's own CLI build, give it
# a placeholder instead of building the UI.
ENV ARGO_VERSION=v4.1.6
ENV ARGO_JOSE_VERSION=v4.1.9
RUN git clone --quiet --depth 1 --branch ${ARGO_VERSION} https://github.com/argoproj/argo-workflows.git /tmp/argo \
    && cd /tmp/argo \
    && mkdir -p ui/dist/app && touch ui/dist/app/placeholder \
    && GOFLAGS=-mod=mod go get github.com/go-jose/go-jose/v4@${ARGO_JOSE_VERSION} \
    && GOFLAGS=-mod=mod go build -trimpath \
        -ldflags "-s -w -X github.com/argoproj/argo-workflows/v4.version=${ARGO_VERSION}" \
        -o /out/argo ./cmd/argo

# Helm, built with this stage's Go for its standard library
ENV HELM_VERSION=v4.3.1
RUN GOBIN=/out go install -trimpath \
    -ldflags "-s -w -X helm.sh/helm/v4/internal/version.version=${HELM_VERSION}" \
    helm.sh/helm/v4/cmd/helm@${HELM_VERSION}

# Runtime stage - Ubuntu with the runtime Ralph, argo, and OpenCode need
FROM ubuntu:24.04

# Apply Ubuntu's security updates and install the system dependencies so base
# packages such as openssl and libssl3t64 carry no known high-severity findings
RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    just \
    unzip \
    net-tools \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# Install Go toolchain
ENV GO_VERSION=1.27.4
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf - \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Go development tools (air for live reload, templ for templates), the Argo
# Workflows CLI, and Helm come from the go-tools stage, built from source at
# the versions pinned there
ENV GOPATH=/root/go
ENV PATH="${GOPATH}/bin:${PATH}"
COPY --from=go-tools /out/ /usr/local/bin/

# Install Bun runtime
ENV BUN_INSTALL=/usr/local/bun
ENV PATH="${BUN_INSTALL}/bin:${PATH}"
RUN curl -fsSL https://bun.sh/install | bash \
    && ln -s ${BUN_INSTALL}/bin/bun /usr/local/bin/bun \
    && ln -s ${BUN_INSTALL}/bin/bunx /usr/local/bin/bunx

# Install OpenCode CLI, pinned so the layer does not freeze at whatever was
# latest when it was first built. Bun never runs a dependency's lifecycle
# scripts, so --trust is required for the package's postinstall to select the
# native opencode binary for this platform; without it the install leaves the
# shebang-less bin/opencode.exe stub that fails as "exec format error".
ENV OPENCODE_VERSION=2.2.7
RUN bun install -g --trust @opencode/cli@${OPENCODE_VERSION} \
    && ln -s ${BUN_INSTALL}/bin/opencode /usr/local/bin/opencode

# Configure OpenCode as an unattended coding agent whose sandbox is this
# container. The config is the machine-wide global file so every run inherits
# it. It allows every tool action, because the agent runs non-interactively
# inside the container and the container boundary is the sandbox: a rule that
# resolved to "ask" would stall a run with no client to answer. OpenCode's own
# agent policies still append after these rules, so built-in agents keep their
# role restrictions. Updates are disabled so the pinned OPENCODE_VERSION is the
# version every run uses.
RUN mkdir -p /root/.config/opencode
COPY container/opencode.json /root/.config/opencode/opencode.json

# Copy ralph binary from builder
COPY --from=builder /build/ralph /usr/local/bin/ralph

# Set up working directory
WORKDIR /workspace

# Default command
CMD ["/bin/sh"]
