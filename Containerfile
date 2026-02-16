# Build stage - compile ralph
FROM docker.io/library/golang:1.25-bookworm AS builder

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and version files
COPY . .

# Build ralph binary using Makefile (includes version injection)
RUN make build

# Runtime stage - use official Playwright image with all browsers pre-installed
FROM mcr.microsoft.com/playwright:v1.58.2-noble

# Install additional system dependencies (Playwright deps already included)
RUN apt-get update && apt-get install -y \
    make \
    unzip \
    net-tools \
    && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Install Go toolchain
ENV GO_VERSION=1.25.0
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf - \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Install Go development tools (air for live reload, templ for templates)
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

# Install OpenCode CLI
RUN bun install -g opencode-ai \
    && ln -s ${BUN_INSTALL}/bin/opencode /usr/local/bin/opencode

# Note: Playwright and all browsers are pre-installed in the base image

# Copy ralph binary from builder
COPY --from=builder /build/ralph /usr/local/bin/ralph

# Set up working directory
WORKDIR /workspace

# Default command
CMD ["/bin/sh"]
