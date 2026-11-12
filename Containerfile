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

# Git stage - Ubuntu's git package depends on the full Perl runtime. Keep the
# package out of the runtime and copy only the compiled git files, which link
# against the shared libraries the runtime installs.
FROM ubuntu:24.04 AS git

RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/*

# Argo stage - build the Argo Workflows CLI from its release tag. Its latest
# release still ships go-jose with a known HIGH vulnerability, and a released
# binary cannot be patched. Building from the release tag lets us raise that
# dependency and keeps the Go module cache out of the runtime image. Drop the
# override once a release ships the fixed version.
FROM docker.io/library/golang:1.27.4-bookworm AS argo

ENV CGO_ENABLED=0
RUN mkdir /out

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

# OpenCode stage - install the pinned OpenCode CLI in a Bun image and keep only
# the native linux/amd64 binary. The module cache and the platform packages for
# musl, baseline, and other operating systems stay in this stage.
FROM --platform=linux/amd64 docker.io/oven/bun:1.4.4-slim AS opencode

ENV BUN_INSTALL=/usr/local/bun
ENV BUN_INSTALL_BIN=${BUN_INSTALL}/bin
ENV PATH="${BUN_INSTALL}/bin:${PATH}"

# Bun never runs a dependency's lifecycle scripts, so --trust is required for
# the package's postinstall to select the native OpenCode binary; without it
# the install leaves the shebang-less bin/opencode.exe stub that fails as
# "exec format error".
ENV OPENCODE_VERSION=2.2.7
RUN bun install -g --trust @opencode/cli@${OPENCODE_VERSION} \
    && mkdir -p /out \
    && cp "$(readlink -f "$(command -v opencode)")" /out/opencode

# Runtime stage - Ubuntu with the runtime Ralph, argo, and OpenCode need
FROM ubuntu:24.04

# Apply Ubuntu's security updates and install the system dependencies so base
# packages such as openssl and libssl3t64 carry no known high-severity findings
RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    libcurl3t64-gnutls \
    libexpat1 \
    libpcre2-8-0 \
    && rm -rf /var/lib/apt/lists/* /usr/share/doc/* /usr/share/info/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod a+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/* /usr/share/doc/* /usr/share/info/*

# Copy only the compiled git commands and templates. The scripts and docs from
# the git package stay in its build stage.
COPY --from=git /usr/bin/git /usr/bin/git-shell /usr/bin/scalar /usr/bin/
COPY --from=git /usr/lib/git-core /usr/lib/git-core
COPY --from=git /usr/share/git-core /usr/share/git-core

# The Argo Workflows CLI comes from the argo stage, built from source at the
# version pinned there
COPY --from=argo /out/argo /usr/local/bin/argo

# Copy only the native OpenCode binary from its build stage. The module cache
# and the platform packages stay out of the runtime.
COPY --from=opencode /out/opencode /usr/local/bin/opencode

# Configure OpenCode as an unattended coding agent whose sandbox is this
# container. The config is the machine-wide global file so every run inherits
# it. It allows every tool action, because the agent runs non-interactively
# inside the container and the container boundary is the sandbox: a rule that
# resolved to "ask" would stall a run with no client to answer. OpenCode's own
# agent policies still append after these rules, so built-in agents keep their
# role restrictions. Updates are disabled so the version pinned in the OpenCode
# build stage is the version every run uses.
RUN mkdir -p /root/.config/opencode
COPY container/opencode.json /root/.config/opencode/opencode.json

# Copy ralph binary from builder
COPY --from=builder /build/ralph /usr/local/bin/ralph

# Set up working directory
WORKDIR /workspace

# Default command
CMD ["/bin/sh"]
