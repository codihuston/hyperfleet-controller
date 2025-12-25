# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bootstrap-service ./cmd/bootstrap-service

# Runtime stage - Use Ubuntu with .NET runtime for GitHub Actions runner compatibility
# Docker will automatically select the appropriate architecture (amd64/arm64)
FROM mcr.microsoft.com/dotnet/runtime:6.0-jammy AS runtime

# Install required packages for GitHub Actions runner
RUN apt-get update && apt-get install -y \
    curl \
    tar \
    gzip \
    bash \
    git \
    docker.io \
    sudo \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Create a user for running the service
RUN useradd -m -s /bin/bash runner && \
    echo 'runner ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Add runner to docker group for Docker-in-Docker
RUN usermod -aG docker runner

WORKDIR /opt/hyperfleet

# Copy the bootstrap service binary
COPY --from=builder /app/bootstrap-service .

# Create config directory and runner directories with proper ownership
RUN mkdir -p /etc/hyperfleet /opt/actions-runner /tmp/runner-work && \
    chown -R runner:runner /opt/actions-runner /tmp/runner-work /etc/hyperfleet /opt/hyperfleet

# Switch to runner user
USER runner

ENTRYPOINT ["./bootstrap-service"]