# HyperFleet Bootstrap Service

The HyperFleet Bootstrap Service is a lightweight binary that runs on ephemeral VMs to handle runner registration and lifecycle management for CI/CD platforms.

## Overview

The bootstrap service is embedded into VM templates and automatically starts when VMs boot. It handles:

1. **Runner Download**: Downloads the appropriate CI/CD runner binary
2. **Runner Configuration**: Configures the runner with registration tokens
3. **Job Execution**: Monitors runner execution
4. **Cleanup**: Cleans up and shuts down the VM after job completion

## Supported Platforms

- **GitHub Actions** (`runner-token` method)
- **SPIFFE/SPIRE** (`join-token` method) - *Coming soon*

## Configuration

The service reads configuration from a JSON file (default: `/etc/hyperfleet/runner-config.json`):

### GitHub Actions Configuration

```json
{
  "method": "runner-token",
  "platform": "github-actions",
  "runner_token": "AABCDEFGHIJKLMNOP...",
  "registration_url": "https://github.com/owner/repo",
  "runner_name": "runner-abc123-def456",
  "labels": ["self-hosted", "hyperfleet", "ephemeral"],
  "expires_at": "2025-12-25T06:00:55.977-06:00",
  "runner": {
    "download_url": "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz",
    "install_path": "/opt/actions-runner",
    "work_dir": "/tmp/runner-work"
  }
}
```

### Configuration Fields

| Field | Description | Default |
|-------|-------------|---------|
| `method` | Attestation method (`runner-token`, `join-token`) | Required |
| `platform` | CI/CD platform (`github-actions`) | Required for `runner-token` |
| `runner_token` | Short-lived registration token | Required for `runner-token` |
| `registration_url` | Platform URL where runner registers | Required |
| `runner_name` | Unique runner name | Required |
| `labels` | Runner labels/tags | `[]` |
| `expires_at` | Token expiration time (RFC3339) | Optional |
| `runner.download_url` | Runner binary download URL | GitHub Actions v2.311.0 |
| `runner.install_path` | Installation directory | `/opt/actions-runner` |
| `runner.work_dir` | Job working directory | `/tmp/runner-work` |

## Usage

### Command Line

```bash
# Use default config path
./bootstrap-service

# Use custom config path
./bootstrap-service --config /path/to/config.json
```

### VM Template Integration

The bootstrap service is typically embedded in VM templates and started via systemd:

```yaml
# systemd service file
[Unit]
Description=HyperFleet VM Bootstrap Service
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
ExecStart=/opt/hyperfleet/bootstrap-service --config /etc/hyperfleet/runner-config.json
Restart=no
User=root

[Install]
WantedBy=multi-user.target
```

## Security Model

### Token Security

- **Registration tokens** are short-lived (1 hour maximum)
- **Registration tokens** have limited scope (runner registration only)
- **Long-lived credentials** (PATs/App keys) never leave Kubernetes
- **VMs are ephemeral** - destroyed after job completion

### Process Flow

1. **HyperFleet Operator** generates registration token using PAT/App credentials
2. **Registration token** injected into VM via cloud-init
3. **Bootstrap service** uses registration token to register runner
4. **Runner executes job** and exits (ephemeral mode)
5. **Bootstrap service** cleans up and shuts down VM

## Building

```bash
# Build the bootstrap service
make build-bootstrap

# Run tests
go test ./cmd/bootstrap-service/
```

## Testing

### Prerequisites

1. **GitHub Personal Access Token**: Create a PAT with `repo` scope (or `public_repo` for public repositories)
2. **Docker**: Required for containerized testing
3. **jq**: Required for JSON processing in test scripts

### Setup

1. Add GitHub configuration to your `.env` file:
```bash
# GitHub Configuration (add to existing .env)
GH_PAT=your_github_personal_access_token
OWNER=your_github_username_or_org
REPO=your_repository_name
```

2. Make sure you have a GitHub repository with Actions enabled

### Running Tests

Execute the test runner:
```bash
./cmd/bootstrap-service/test-runner.sh
```

This will:
1. Generate a GitHub runner registration token
2. Build a Docker image with the bootstrap service
3. Run the service in a container
4. Download and configure the GitHub Actions runner
5. Wait for workflow jobs (you can trigger a workflow to test)
6. Clean up automatically

### Manual Testing

You can also test individual components:

1. **Generate configuration**:
```bash
./cmd/bootstrap-service/test-config.sh
```

2. **Build Docker image**:
```bash
docker build -f cmd/bootstrap-service/Dockerfile -t hyperfleet-bootstrap:test .
```

3. **Run container**:
```bash
docker run --rm \
    -v /tmp/runner-config.json:/etc/hyperfleet/runner-config.json:ro \
    hyperfleet-bootstrap:test \
    --config /etc/hyperfleet/runner-config.json
```

## Deployment Strategies

### 1. Embedded in VM Template (Recommended)

```bash
# During VM template creation
sudo cp bootstrap-service /opt/hyperfleet/bootstrap-service
sudo chmod +x /opt/hyperfleet/bootstrap-service
```

### 2. Cloud-Init Download

```yaml
write_files:
  - path: /opt/hyperfleet/download-bootstrap.sh
    content: |
      #!/bin/bash
      curl -L -o /opt/hyperfleet/bootstrap-service \
        "https://releases.example.com/bootstrap-service-linux-amd64"
      chmod +x /opt/hyperfleet/bootstrap-service
    permissions: '0755'

runcmd:
  - /opt/hyperfleet/download-bootstrap.sh
  - /opt/hyperfleet/bootstrap-service --config /etc/hyperfleet/runner-config.json
```

### 3. Container Image

```bash
docker run --rm -v /etc/hyperfleet:/config \
  hyperfleet/bootstrap-service:latest \
  --config /config/runner-config.json
```

## Troubleshooting

### Common Issues

1. **Network connectivity**: Ensure VM can reach GitHub/platform APIs
2. **Token expiration**: Registration tokens expire in 1 hour
3. **Permissions**: Bootstrap service needs root access for VM shutdown
4. **Disk space**: Ensure sufficient space for runner download and jobs

### Logging

The bootstrap service logs to stdout with timestamps:

```
[github-bootstrap] 2025/12/25 05:31:58 Starting GitHub runner bootstrap for runner-abc123
[github-bootstrap] 2025/12/25 05:31:58 Downloading GitHub Actions runner from https://...
[github-bootstrap] 2025/12/25 05:32:15 Configuring runner runner-abc123
[github-bootstrap] 2025/12/25 05:32:20 Starting GitHub Actions runner
[github-bootstrap] 2025/12/25 05:45:30 Runner completed, initiating VM shutdown
```

### Exit Codes

- `0`: Success (normal completion)
- `1`: Configuration error
- `2`: Download/installation error
- `3`: Runner configuration error
- `4`: Runner execution error

## Architecture

The bootstrap service follows a simple pipeline:

```
Load Config → Download Runner → Configure Runner → Execute Jobs → Cleanup → Shutdown
```

Each step is designed to be:
- **Idempotent**: Safe to retry
- **Observable**: Clear logging at each step
- **Fail-fast**: Exit immediately on errors
- **Secure**: No persistent secrets storage

## Future Enhancements

- **SPIFFE/SPIRE support** for `join-token` method
- **GitLab CI support** for GitLab runners
- **Drone CI support** for Drone agents
- **Multi-architecture binaries** (ARM64, etc.)
- **Metrics collection** for observability