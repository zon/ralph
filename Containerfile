# Build stage - compile ralph
FROM docker.io/library/golang:1.26-bookworm AS builder

WORKDIR /build

RUN curl -fsSL https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and version files
COPY . .

# Build ralph binary using justfile (includes version injection)
RUN just build

# Runtime stage - use official Playwright image with all browsers pre-installed
FROM mcr.microsoft.com/playwright:v1.63.0-noble

# Apply Ubuntu's security updates and install additional system dependencies
# (Playwright deps already included) so base packages such as openssl and
# libssl3t64 carry no known high-severity findings
RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y \
    just \
    unzip \
    net-tools \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Upgrade npm so its bundled dependencies carry no known vulnerabilities
ENV NPM_VERSION=12.2.1
RUN npm install -g npm@${NPM_VERSION}

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Install Go toolchain
ENV GO_VERSION=1.26.7
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf - \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Install Go development tools (air for live reload, templ for templates)
ENV GOPATH=/root/go
ENV PATH="${GOPATH}/bin:${PATH}"
RUN go install github.com/air-verse/air@latest \
    && go install github.com/a-h/templ/cmd/templ@latest \
    && ln -s /root/go/bin/air /usr/local/bin/air \
    && ln -s /root/go/bin/templ /usr/local/bin/templ

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
ENV OPENCODE_VERSION=2.0.29
RUN bun install -g --trust @opencode/cli@${OPENCODE_VERSION} \
    && ln -s ${BUN_INSTALL}/bin/opencode /usr/local/bin/opencode

# Install Argo Workflows CLI
ENV ARGO_VERSION=v4.1.6
RUN curl -sLO "https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-linux-amd64.gz" \
    && gunzip argo-linux-amd64.gz \
    && chmod +x argo-linux-amd64 \
    && mv argo-linux-amd64 /usr/local/bin/argo

# Install Helm
ENV HELM_VERSION=v4.3.1
RUN curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz" | tar -xz \
    && mv linux-amd64/helm /usr/local/bin/helm \
    && rm -rf linux-amd64

# Note: Playwright and all browsers are pre-installed in the base image

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
