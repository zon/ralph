# Build stage - compile ralph
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build ralph binary
RUN CGO_ENABLED=0 GOOS=linux go build -o ralph .

# Runtime stage - multi-purpose image with all dependencies
FROM ubuntu:24.04

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    openssh-client \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Install Go toolchain
ENV GO_VERSION=1.25.0
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf - \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Install Bun runtime
ENV BUN_INSTALL=/usr/local/bun
RUN curl -fsSL https://bun.sh/install | bash \
    && ln -s ${BUN_INSTALL}/bin/bun /usr/local/bin/bun

# Install Playwright dependencies
RUN apt-get update && apt-get install -y \
    # Playwright system dependencies
    libnss3 \
    libnspr4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libdbus-1-3 \
    libxkbcommon0 \
    libatspi2.0-0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2 \
    libxshmfence1 \
    && rm -rf /var/lib/apt/lists/*

# Install Playwright via bun
RUN bun add -g playwright \
    && bunx playwright install chromium \
    && bunx playwright install firefox \
    && bunx playwright install webkit

# Copy ralph binary from builder
COPY --from=builder /build/ralph /usr/local/bin/ralph

# Set up working directory
WORKDIR /workspace

# Default command
CMD ["/bin/sh"]
