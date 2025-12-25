# HyperFleet Operator

A Kubernetes operator for managing virtual machine lifecycle on proxmox, providing automated ephemeral VM provisioning and management for dynamic workloads like GitHub Actions runners.

While only proxmox is currently supported, the
architectural design attempts to be flexible enough
to support any on-premise hypervisor that can be
managed with an API in the future.

## Description

The HyperFleet Operator enables Kubernetes-native management of ephemereal virtual machines on proxmox. It provides Custom Resource Definitions (CRDs) for defining hypervisor clusters, VM templates, and VM pools, with automated provisioning, scaling, and lifecycle management.

**Current Features:**
- **HypervisorCluster**: Connect and manage Proxmox VE clusters
- **Connection Validation**: Automatic testing of hypervisor connectivity and credentials
- **TLS Support**: Handles self-signed certificates for development environments
- **Kubernetes-native**: Full integration with Kubernetes RBAC, secrets, and status reporting

**Planned Features:**
- VM template management and validation
- Automated VM provisioning and cleanup
- GitHub Actions runner integration
- Multi-hypervisor support (VMware, Hyper-V)
- Advanced scaling policies and lifecycle management

## Quick Start

For development and testing, see the [Contributing Guide](CONTRIBUTING.md) for complete setup instructions.

## Documentation

- **[Contributing Guide](CONTRIBUTING.md)** - Complete development workflow and testing
- **[API Reference](api/v1alpha1/)** - Custom Resource Definitions
- **[Specifications](.kiro/specs/)** - Design documents and requirements

## Architecture

The operator follows standard Kubernetes operator patterns with:

- **Custom Resources**: Define desired state for hypervisor infrastructure
- **Controllers**: Reconcile actual state with desired state
- **Provider Interface**: Pluggable hypervisor implementations
- **Status Reporting**: Detailed condition reporting and event emission

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
