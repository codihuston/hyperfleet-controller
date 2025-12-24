# Requirements Document

## Introduction

HyperFleet is a Kubernetes-native control plane for provisioning and lifecycle-managing ephemeral virtual machines on external hypervisors, starting with Proxmox VE. It enables VM-per-job, autoscaled GitHub Actions runners and similar workloads without relying on cloud provider infrastructure, providing a declarative, GitOps-friendly way to elastically provision short-lived VMs on on-prem or self-hosted hypervisors.

## Glossary

- **HyperFleet_Operator**: The Kubernetes operator that manages the lifecycle of ephemeral VMs
- **Proxmox_VE**: The primary hypervisor platform for VM execution
- **MachineClaim**: A Kubernetes custom resource representing a desired VM instance
- **HypervisorCluster**: A custom resource defining connection details for a hypervisor environment
- **RunnerPool**: A custom resource defining workload intent and scaling policy
- **Bootstrap_Service**: The service that provides short-lived credentials and configuration to VMs
- **VM_Template**: A pre-baked virtual machine image used for cloning
- **SPIFFE**: Secure Production Identity Framework for Everyone, used for VM identity
- **Attestation_Method**: The mechanism used to verify VM identity (join token, TPM)
- **TPM**: Trusted Platform Module, hardware-based security for identity attestation

## Requirements

### Requirement 1: Hypervisor Management

**User Story:** As a platform engineer, I want to define and manage hypervisor clusters, so that I can provision VMs across multiple Proxmox environments.

#### Acceptance Criteria

1. WHEN a HypervisorCluster resource is created, THE HyperFleet_Operator SHALL validate the connection parameters and establish connectivity
2. WHEN connecting to Proxmox VE, THE HyperFleet_Operator SHALL authenticate using the provided credentials and verify API access
3. WHEN a HypervisorCluster becomes unavailable, THE HyperFleet_Operator SHALL update the resource status and prevent new VM provisioning
4. THE HyperFleet_Operator SHALL support multiple HypervisorCluster resources for multi-environment deployments
5. WHEN hypervisor credentials are rotated, THE HyperFleet_Operator SHALL detect the change and re-establish connections

### Requirement 2: VM Template Management

**User Story:** As a platform engineer, I want to define reusable VM templates, so that I can standardize VM configurations across different workloads.

#### Acceptance Criteria

1. WHEN a HypervisorMachineTemplate is created, THE HyperFleet_Operator SHALL validate the template specification against the target hypervisor
2. THE HyperFleet_Operator SHALL support referencing Proxmox VM templates by ID or name
3. WHEN a template specifies cloud-init configuration, THE HyperFleet_Operator SHALL merge it with runtime bootstrap data
4. THE HyperFleet_Operator SHALL validate CPU, memory, and disk specifications against hypervisor capabilities
5. WHEN a referenced template is deleted from Proxmox, THE HyperFleet_Operator SHALL update the template status to reflect unavailability

### Requirement 3: VM Lifecycle Management

**User Story:** As a platform engineer, I want to provision and manage ephemeral VMs, so that I can provide on-demand compute resources for CI/CD workloads.

#### Acceptance Criteria

1. WHEN a MachineClaim is created, THE HyperFleet_Operator SHALL provision a VM from the specified template
2. WHEN provisioning a VM, THE HyperFleet_Operator SHALL inject bootstrap credentials via cloud-init
3. WHEN a VM is successfully created, THE HyperFleet_Operator SHALL update the MachineClaim status to "Provisioning"
4. WHEN a VM completes bootstrap, THE HyperFleet_Operator SHALL update the MachineClaim status to "Ready"
5. WHEN a MachineClaim is deleted, THE HyperFleet_Operator SHALL destroy the associated VM and clean up resources
6. THE HyperFleet_Operator SHALL use finalizers to ensure proper cleanup before MachineClaim deletion
7. WHEN a VM fails to provision, THE HyperFleet_Operator SHALL retry according to configured backoff policy
8. WHEN a VM shuts down (either self-terminated or stopped), THE HyperFleet_Operator SHALL detect the shutdown state and automatically delete the VM from the hypervisor within 5 minutes
9. THE HyperFleet_Operator SHALL periodically reconcile VM states and clean up any VMs that are stopped but not yet deleted
10. WHEN a GitHub Actions runner completes a job and shuts down the VM, THE HyperFleet_Operator SHALL detect the VM shutdown state and remove both the VM from Proxmox and the corresponding MachineClaim from Kubernetes
11. THE HyperFleet_Operator SHALL implement a VM state monitoring loop that checks VM power states every 30 seconds and triggers cleanup for any stopped VMs

### Requirement 4: Runner Pool Scaling

**User Story:** As a DevOps engineer, I want to define runner pools with scaling policies, so that I can automatically provision VMs based on workload demand.

#### Acceptance Criteria

1. WHEN a RunnerPool is created, THE HyperFleet_Operator SHALL create MachineClaims according to the minimum replica count
2. WHEN scaling up a RunnerPool, THE HyperFleet_Operator SHALL create additional MachineClaims up to the maximum limit
3. WHEN scaling down a RunnerPool, THE HyperFleet_Operator SHALL gracefully drain and delete excess MachineClaims
4. THE HyperFleet_Operator SHALL respect VM lifecycle policies including TTL and maximum lifetime
5. WHEN a VM exceeds its maximum lifetime, THE HyperFleet_Operator SHALL terminate it and provision a replacement

### Requirement 5: CI/CD Platform Integration

**User Story:** As a developer, I want VMs to automatically register as CI/CD runners for various platforms, so that my jobs can execute on fresh, ephemeral compute resources.

#### Acceptance Criteria

1. WHEN a VM boots successfully, THE Bootstrap_Service SHALL provide runner registration credentials for the configured CI/CD platform
2. WHEN registering with a CI/CD platform, THE VM SHALL use the provided configuration (labels, tags, queues) from the RunnerPool specification
3. WHEN a CI job completes, THE VM SHALL self-terminate and deregister from the CI/CD platform
4. THE HyperFleet_Operator SHALL support webhook-based autoscaling triggered by platform-specific events (GitHub workflow_job, GitLab pipeline, etc.)
5. WHEN no jobs are available, THE VM SHALL wait for a configurable timeout before self-terminating
6. THE HyperFleet_Operator SHALL support multiple CI/CD platforms including GitHub Actions, GitLab CI, and Buildkite through a pluggable interface

### Requirement 6: Security and Identity Management

**User Story:** As a security engineer, I want VMs to use multiple attestation methods and secure identity mechanisms, so that I can choose the appropriate security model for different environments and compliance requirements.

#### Acceptance Criteria

1. WHEN provisioning a VM, THE HyperFleet_Operator SHALL support multiple attestation methods including join tokens and TPM attestation
2. THE Bootstrap_Service SHALL provide a pluggable attestation interface supporting join tokens as the default method
3. WHEN a VM requests credentials, THE Bootstrap_Service SHALL validate the VM's identity using the configured attestation method before providing GitHub runner registration tokens
4. THE HyperFleet_Operator SHALL ensure no long-lived GitHub tokens or cluster secrets are stored on VM disks regardless of attestation method
5. WHEN bootstrap credentials expire, THE VM SHALL be unable to obtain new credentials and SHALL self-terminate
6. THE HyperFleet_Operator SHALL support TPM-based attestation for environments requiring hardware-backed identity
7. THE HyperFleet_Operator SHALL only manage VMs that it has created and tagged with HyperFleet ownership metadata
8. WHEN performing VM operations, THE HyperFleet_Operator SHALL validate VM ownership before allowing any lifecycle operations

### Requirement 7: Operator Framework Implementation

**User Story:** As a platform engineer, I want the operator built with standard Kubernetes patterns, so that it integrates well with existing cluster management tools.

#### Acceptance Criteria

1. THE HyperFleet_Operator SHALL be implemented using the Operator SDK and controller-runtime framework
2. THE HyperFleet_Operator SHALL use reconciliation loops for all custom resource management
3. WHEN custom resources are created or modified, THE HyperFleet_Operator SHALL reconcile to the desired state
4. THE HyperFleet_Operator SHALL implement proper RBAC with least-privilege access patterns
5. THE HyperFleet_Operator SHALL support installation via Kustomize manifests and Helm charts, with future OLM (Operator Lifecycle Manager) support
6. THE HyperFleet_Operator SHALL emit Kubernetes events for important lifecycle transitions
7. THE HyperFleet_Operator SHALL expose Prometheus metrics for monitoring and alerting
8. THE HyperFleet_Operator SHALL run as rootless containers with non-privileged security contexts

### Requirement 8: Proxmox VE Integration

**User Story:** As a platform engineer, I want seamless integration with Proxmox VE, so that I can leverage existing hypervisor infrastructure for VM provisioning.

#### Acceptance Criteria

1. THE HyperFleet_Operator SHALL use the Proxmox VE REST API for all VM operations
2. WHEN cloning VMs, THE HyperFleet_Operator SHALL use Proxmox linked clones for fast provisioning
3. THE HyperFleet_Operator SHALL tag VMs with ownership metadata for garbage collection and auditing
4. WHEN injecting cloud-init data, THE HyperFleet_Operator SHALL use Proxmox cloud-init drive functionality
5. THE HyperFleet_Operator SHALL support Proxmox storage pools and network bridge configuration
6. WHEN VMs are destroyed, THE HyperFleet_Operator SHALL ensure complete cleanup of VM resources and metadata
7. THE HyperFleet_Operator SHALL only perform operations on VMs that contain HyperFleet ownership tags
8. WHEN discovering existing VMs, THE HyperFleet_Operator SHALL ignore VMs without proper ownership metadata

### Requirement 9: Configuration and Observability

**User Story:** As a platform operator, I want comprehensive configuration options and observability, so that I can effectively manage and troubleshoot the system.

#### Acceptance Criteria

1. THE HyperFleet_Operator SHALL support configuration via ConfigMaps and environment variables
2. THE HyperFleet_Operator SHALL log structured events for all major operations and state transitions
3. WHEN errors occur, THE HyperFleet_Operator SHALL provide detailed error messages in resource status conditions
4. THE HyperFleet_Operator SHALL expose metrics for VM provisioning time, success rate, and resource utilization
5. THE HyperFleet_Operator SHALL support external secret management via ExternalSecrets or similar operators
6. THE HyperFleet_Operator SHALL validate all custom resource specifications and provide clear validation errors

### Requirement 10: Logging and Observability

**User Story:** As a platform operator, I want centralized logging with configurable levels and distributed tracing, so that I can effectively monitor, debug, and troubleshoot the operator across different environments.

#### Acceptance Criteria

1. THE HyperFleet_Operator SHALL implement centralized structured logging using a standard logging library (logrus, zap, or slog)
2. THE HyperFleet_Operator SHALL support configurable log levels (DEBUG, INFO, WARN, ERROR) via the LOG_LEVEL environment variable
3. WHEN LOG_LEVEL is not set, THE HyperFleet_Operator SHALL default to INFO level logging
4. THE HyperFleet_Operator SHALL log all major operations including VM provisioning, deletion, status changes, and error conditions with appropriate context
5. THE HyperFleet_Operator SHALL implement OpenTelemetry tracing for distributed observability across all major operations
6. WHEN processing reconciliation loops, THE HyperFleet_Operator SHALL create trace spans for each major operation with relevant attributes
7. THE HyperFleet_Operator SHALL support configurable OpenTelemetry exporters via environment variables (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME)
8. THE HyperFleet_Operator SHALL include trace correlation IDs in log messages to connect logs with distributed traces
9. THE HyperFleet_Operator SHALL emit custom metrics and traces for VM provisioning time, API call latency, and error rates
10. WHEN errors occur, THE HyperFleet_Operator SHALL log stack traces at DEBUG level and error summaries at ERROR level with trace context

### Requirement 11: Extensibility and Future Hypervisor Support

**User Story:** As a platform architect, I want the operator designed for extensibility, so that I can add support for additional hypervisor platforms like Hyper-V and VMware in the future while maintaining a consistent Kubernetes API.

#### Acceptance Criteria

1. THE HyperFleet_Operator SHALL implement a provider interface that abstracts hypervisor-specific operations for Proxmox VE, Hyper-V, VMware, and other platforms
2. THE Proxmox_Provider SHALL implement all required provider interface methods as the initial reference implementation
3. THE HyperFleet_Operator SHALL support provider selection via HypervisorCluster configuration with provider-specific parameters
4. WHEN adding new hypervisor providers, THE HyperFleet_Operator SHALL maintain backward compatibility with existing Proxmox resources
5. THE provider interface SHALL support common operations including VM creation, deletion, status checking, metadata tagging, and template management across different hypervisor platforms