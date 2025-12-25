# HyperFleet Operator Development Guide

This guide covers the complete development workflow for the HyperFleet Operator.

## Overview

The HyperFleet Operator manages virtual machine lifecycle on hypervisor platforms, starting with Proxmox VE support.

## Development Workflow

### 1. Prerequisites

- Go 1.21+
- Docker Desktop with Kubernetes enabled
- kubectl configured for your cluster
- Access to a Proxmox VE server (for testing)

### 2. Project Setup

```bash
# Clone the repository
git clone <repository-url>
cd hyperfleet-operator

# Install dependencies
go mod download

# Generate code and manifests
make generate manifests
```

### 3. Install CRDs

Install the Custom Resource Definitions into your cluster:

```bash
make install
```

This installs the following CRDs:
- `HypervisorCluster` - Represents a hypervisor cluster connection
- `HypervisorMachineTemplate` - VM template definitions
- `MachineClaim` - Individual VM requests
- `RunnerPool` - Managed pools of VMs

### 4. Configure Credentials

Create Proxmox API credentials secret:

```bash
kubectl create secret generic test-proxmox-credentials \
  --from-literal=tokenId="your-token-id" \
  --from-literal=tokenSecret="your-token-secret"
```

### 5. Configure Environment

Copy and edit the environment configuration:

```bash
cp .env.example .env
# Edit .env with your Proxmox details
```

### 6. Run the Controller

Start the controller locally for development:

```bash
make run
```

The controller will:
- Connect to your Kubernetes cluster
- Watch for HypervisorCluster resources
- Validate connections to Proxmox servers
- Update resource status with connection results

### 7. Apply Sample Configuration

In a separate terminal, apply the sample HypervisorCluster:

```bash
./scripts/apply-sample.sh
```

### 8. Verify Operation

Check that everything is working:

```bash
# Check HypervisorCluster status
kubectl get hypervisorclusters

# Get detailed information
kubectl describe hypervisorcluster proxmox-test

# View controller logs
# (check the terminal where 'make run' is running)
```

## Configuration Reference

### Environment Variables (.env)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CLUSTER_NAME` | Name of the HypervisorCluster resource | `proxmox-test` | No |
| `NAMESPACE` | Kubernetes namespace | `default` | No |
| `PROXMOX_ENDPOINT` | Proxmox VE API endpoint (include `/api2/json`) | - | **Yes** |
| `NODE_1` | First Proxmox node name | `pve-node-1` | No |
| `NODE_2` | Second Proxmox node name | `pve-node-2` | No |
| `DEFAULT_STORAGE` | Default storage pool | `local-lvm` | No |
| `DEFAULT_NETWORK` | Default network bridge | `vmbr0` | No |
| `DNS_DOMAIN` | DNS domain for VMs | `hyperfleet.local` | No |
| `DNS_SERVER_1` | Primary DNS server | `192.168.1.1` | No |
| `DNS_SERVER_2` | Secondary DNS server | `8.8.8.8` | No |
| `SECRET_NAME` | Name of the credentials secret | `test-proxmox-credentials` | No |
| `ENVIRONMENT` | Environment tag | `test` | No |

### Example .env Configuration

```bash
# HyperFleet Operator Development Configuration
CLUSTER_NAME=my-proxmox-dev
NAMESPACE=default
PROXMOX_ENDPOINT=https://pve.example.com:8006/api2/json
NODE_1=pve-01
DEFAULT_STORAGE=local-lvm
DEFAULT_NETWORK=vmbr0
DNS_DOMAIN=dev.example.com
DNS_SERVER_1=192.168.1.1
DNS_SERVER_2=8.8.8.8
SECRET_NAME=test-proxmox-credentials
ENVIRONMENT=development
```

## Development Commands

### Code Generation

```bash
# Generate deepcopy methods and CRD manifests
make generate manifests

# Update API documentation
make api-docs
```

### Testing

```bash
# Run unit tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests (requires running cluster)
make test-integration

# Run end-to-end tests
make test-e2e
```

### Building

```bash
# Build the manager binary
make build

# Build and push container image
make docker-build docker-push IMG=<registry>/hyperfleet-operator:tag
```

### Deployment

```bash
# Deploy to cluster (for production testing)
make deploy IMG=<registry>/hyperfleet-operator:tag

# Undeploy from cluster
make undeploy
```

## Troubleshooting

### Controller Issues

1. **Controller won't start:**
   ```bash
   # Check if CRDs are installed
   kubectl get crd | grep hypervisor
   
   # Reinstall CRDs if missing
   make install
   ```

2. **Permission errors:**
   ```bash
   # Check your kubeconfig
   kubectl auth can-i create hypervisorclusters
   
   # Verify cluster connection
   kubectl cluster-info
   ```

### Proxmox Connection Issues

1. **TLS certificate errors:**
   - The controller uses `InsecureSkipVerify: true` for development
   - Ensure your Proxmox endpoint includes `/api2/json`

2. **Authentication failures:**
   ```bash
   # Verify secret exists and has correct keys
   kubectl get secret test-proxmox-credentials -o yaml
   
   # Test Proxmox API manually
   curl -k "https://your-proxmox:8006/api2/json/version"
   ```

3. **Network connectivity:**
   ```bash
   # Test from your development machine
   curl -k "https://your-proxmox:8006/api2/json/version"
   
   # Check if Proxmox is accessible from cluster
   kubectl run test-pod --image=curlimages/curl --rm -it -- \
     curl -k "https://your-proxmox:8006/api2/json/version"
   ```

### Resource Status Issues

1. **HypervisorCluster shows Not Ready:**
   ```bash
   # Check detailed status
   kubectl describe hypervisorcluster <name>
   
   # Look for error messages in status conditions
   kubectl get hypervisorcluster <name> -o yaml
   ```

2. **Controller logs show errors:**
   - Check the terminal where `make run` is running
   - Look for connection errors, authentication failures, or API issues

## Manual Testing Workflow

For manual testing without the script:

```bash
# 1. Load environment variables
export $(grep -v '^#' .env | xargs)

# 2. Set defaults for optional variables
export CLUSTER_NAME=${CLUSTER_NAME:-proxmox-test}
export NAMESPACE=${NAMESPACE:-default}
# ... (set other defaults as needed)

# 3. Apply with environment substitution
envsubst < config/samples/hypervisor_v1alpha1_hypervisorcluster_template.yaml | kubectl apply -f -

# 4. Monitor the resource
kubectl get hypervisorclusters -w
```

## Next Steps

Once you have the HypervisorCluster working:

1. **Implement additional CRDs** (HypervisorMachineTemplate, MachineClaim, RunnerPool)
2. **Add VM provisioning logic** to the controllers
3. **Integrate with GitHub Actions** for runner management
4. **Add comprehensive testing** for all scenarios

## Contributing

1. Follow the conventional commit format
2. Run tests before committing: `make test`
3. Update documentation for API changes
4. Test with real Proxmox infrastructure when possible

For more details, see the project's main README.md and the specifications in `.kiro/specs/`.