# Implementation Plan: HyperFleet Operator

## Overview

This implementation plan converts the HyperFleet operator design into a series of incremental Go development tasks using the Operator SDK and controller-runtime framework. The implementation follows a bottom-up approach, starting with core data structures and providers, then building controllers, and finally integrating the complete system with SPIFFE identity management and GitHub Actions.

## Tasks

- [ ] 1. Project Setup and Core Infrastructure
  - Initialize Go module with Operator SDK
  - Set up project structure with controller-runtime
  - Configure build system, Dockerfile, and Makefile
  - Set up testing framework with gopter for property-based testing
  - _Requirements: Requirement 7 (criteria 1, 2)_

- [ ] 2. Define Custom Resource Definitions (CRDs)
  - [X] 2.1 Create HypervisorCluster CRD and Go types
    - Define HypervisorClusterSpec and HypervisorClusterStatus structs
    - Implement validation webhooks for connection parameters
    - Add kubebuilder markers for OpenAPI schema generation
    - _Requirements: Requirement 1 (criteria 1), Requirement 9 (criteria 6)_

  - [ ]* 2.2 Write property test for HypervisorCluster validation
    - **Property 1: Hypervisor Connection Management**
    - **Validates: Requirement 1 (criteria 1, 3, 5)**

  - [ ] 2.3 Create HypervisorMachineTemplate CRD and Go types
    - Define template specification structures
    - Implement resource validation (CPU, memory, disk)
    - Add cloud-init configuration support
    - Add runner-token workload method for GitHub
    - Implement dynamic GitHub registration token generation
    - Add configurable HTTP client timeout (default 30s)
    - Support repository and organization-level runners (repo takes precedence)
    - Add bootstrap service binary embedding in cloud-init
    - Support independent SPIFFE attestation configuration
    - Add configurable OS/arch support for GitHub runner downloads
    - Make GitHub runner script paths configurable (config.sh, run.sh)
    - _Requirements: 2.1, 2.4_

  - [ ]* 2.4 Write property test for template validation
    - **Property 3: Template Validation and Management**
    - **Validates: Requirements 2.1, 2.4, 2.5, 9.6**

  - [ ] 2.5 Create RunnerPool CRD and Go types
    - Define scaling policies and workload configuration
    - Implement lifecycle policy structures
    - Add GitHub Actions specific configuration
    - _Requirements: 4.1, 4.4_

  - [ ] 2.6 Create MachineClaim CRD and Go types
    - Define VM specification and status structures
    - Implement finalizer support for cleanup
    - Add condition management for status tracking
    - _Requirements: 3.1, 3.5, 3.6_

- [ ] 3. Implement Provider Interface and Proxmox Provider
  - [ ] 3.1 Define provider interface abstraction
    - Create Provider interface with VM lifecycle methods
    - Define common data structures (VMSpec, VMInfo, VMStatus)
    - Implement provider factory pattern
    - _Requirements: 10.1, 10.2, 10.5_

  - [ ]* 3.2 Write property test for provider interface
    - **Property 24: Provider Interface Implementation**
    - **Validates: Requirements 10.2, 10.5**

  - [ ] 3.3 Implement Proxmox VE provider
    - Create ProxmoxProvider struct with REST API client
    - Implement VM creation using linked clones
    - Add cloud-init injection via Proxmox drives
    - Implement VM tagging and metadata management
    - Add VM ownership validation for all operations
    - _Requirements: Requirement 8 (criteria 1, 2, 3, 4, 7, 8)_

  - [ ]* 3.4 Write property test for Proxmox operations
    - **Property 19: Proxmox API Integration**
    - **Validates: Requirement 8 (criteria 1, 2, 4)**

  - [ ]* 3.5 Write property test for VM ownership
    - **Property 26: VM Ownership and Isolation**
    - **Validates: Requirement 6 (criteria 7, 8), Requirement 8 (criteria 7, 8)**

  - [ ] 3.6 Add storage and network configuration support
    - Implement Proxmox storage pool selection
    - Add network bridge configuration
    - Support multiple node selection
    - _Requirements: 8.5_

  - [ ]* 3.7 Write property test for storage and network config
    - **Property 21: Storage and Network Configuration**
    - **Validates: Requirements 8.5**

- [ ] 4. Checkpoint - Core Infrastructure Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Implement Generic Attestation and Bootstrap Token Management
  - [ ] 5.1 Create SPIFFE attestation method interface and implementations
    - Define SPIFFEMethod interface for extensibility (independent of workload)
    - Implement JoinTokenAttestation as default method
    - Add TPMAttestation implementation
    - Create SPIFFE attestation method factory and configuration
    - _Requirements: 6.1, 6.2, 6.6_

  - [ ]* 5.2 Write property test for SPIFFE attestation
    - **Property 14: SPIFFE Attestation and Identity Management**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.5, 6.6**

  - [ ] 5.3 Implement SPIRE integration for SPIFFE attestation methods
    - Add SPIRE server client for join token generation
    - Implement TPM-based SPIRE attestation
    - Create unified SPIFFE identity validation
    - _Requirements: 6.1, 6.2, 6.6_

  - [ ] 5.4 Implement workload credential management (independent of SPIFFE)
    - Add cloud-init template rendering with workload config
    - Implement secure token/certificate generation and storage
    - Add credential expiration and cleanup for all methods
    - Support both SPIFFE-based and direct token methods
    - _Requirements: 3.2, 6.1_

- [ ] 6. Implement Core Controllers
  - [ ] 6.1 Create HypervisorCluster controller
    - Implement reconciliation loop for cluster management
    - Add connection validation and status updates
    - Handle credential rotation and reconnection
    - Implement multi-cluster support
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_
    - **TECHNICAL DEBT**: Add CA certificate trust support for Proxmox TLS connections instead of using InsecureSkipVerify. Allow users to provide CA certificates via Kubernetes secrets for proper TLS validation while maintaining backward compatibility.

  - [ ]* 6.2 Write property test for cluster controller
    - **Property 2: Multi-Hypervisor Support**
    - **Validates: Requirements 1.4**

  - [ ] 6.3 Create HypervisorMachineTemplate controller
    - Implement template validation against hypervisor
    - Add template reference resolution (ID and name)
    - Handle template availability monitoring
    - _Requirements: 2.1, 2.2, 2.5_

  - [ ]* 6.4 Write property test for template controller
    - **Property 4: Template Reference Resolution**
    - **Validates: Requirements 2.2**

  - [ ] 6.5 Create MachineClaim controller
    - Implement VM provisioning workflow
    - Add status condition management (Provisioning → Ready)
    - Implement finalizer-based cleanup
    - Add retry logic with exponential backoff
    - Add VM shutdown detection and automatic cleanup
    - Implement periodic VM state monitoring (every 30 seconds)
    - _Requirements: 3.1, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 3.10, 3.11_

  - [ ]* 6.6 Write property test for MachineClaim lifecycle
    - **Property 6: VM Lifecycle Management**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4**

  - [ ]* 6.7 Write property test for cleanup with finalizers
    - **Property 7: Resource Cleanup with Finalizers**
    - **Validates: Requirements 3.5, 3.6, 8.6**

  - [ ]* 6.8 Write property test for VM shutdown cleanup
    - **Property 27: VM Shutdown Detection and Cleanup**
    - **Validates: Requirements 3.8, 3.9, 3.10, 3.11**

- [ ] 7. Implement RunnerPool Controller and Scaling Logic
  - [ ] 7.1 Create RunnerPool controller
    - Implement replica management (min/max scaling)
    - Add MachineClaim creation and deletion logic
    - Implement lifecycle policy enforcement (TTL, max lifetime)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [ ]* 7.2 Write property test for scaling behavior
    - **Property 9: RunnerPool Scaling Behavior**
    - **Validates: Requirements 4.1, 4.2, 4.3**

  - [ ]* 7.3 Write property test for lifecycle policies
    - **Property 10: VM Lifecycle Policy Enforcement**
    - **Validates: Requirements 4.4, 4.5**

  - [ ] 7.4 Implement GitHub webhook autoscaling
    - Add webhook server for workflow_job events
    - Implement scaling decision logic based on queue depth
    - Add integration with RunnerPool scaling
    - _Requirements: 5.4_

  - [ ]* 7.5 Write property test for webhook scaling
    - **Property 13: Webhook-Based Autoscaling**
    - **Validates: Requirements 5.4**

- [ ] 8. Checkpoint - Controllers Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 9. Implement VM Bootstrap Service (for VM images)
  - [x] 9.1 Create GitHub Actions bootstrap service binary
    - Implement GitHub Actions runner download and configuration using HTTP client
    - Add registration token-based runner setup (ephemeral mode)
    - Implement runner monitoring and auto-termination
    - Add VM shutdown after job completion
    - Support configurable runner labels and names
    - Add configurable OS/arch support with environment variable fallbacks
    - Make GitHub runner script paths configurable (config.sh, run.sh)
    - Support independent SPIFFE attestation alongside runner token method
    - _Requirements: 5.1, 5.2, 5.3, 5.5, 6.6_

  - [ ]* 9.2 Write property test for GitHub integration
    - **Property 11: GitHub Integration via VM Bootstrap Service**
    - **Validates: Requirements 5.1, 5.3, 5.5**

  - [ ]* 9.3 Write property test for runner configuration
    - **Property 12: GitHub Runner Configuration**
    - **Validates: Requirements 5.2**

  - [ ] 9.4 Add security validation for VM bootstrap
    - Implement checks to ensure no persistent secrets on disk
    - Add credential expiration handling
    - Implement secure cleanup on termination
    - _Requirements: 6.4, 6.5_

  - [ ]* 9.5 Write property test for security requirements
    - **Property 15: Security - No Persistent Secrets**
    - **Validates: Requirements 6.4**

- [ ] 10. Implement Observability and Configuration
  - [ ] 10.1 Add Prometheus metrics
    - Implement VM provisioning time metrics
    - Add success rate and resource utilization metrics
    - Create controller performance metrics
    - _Requirements: 7.7, 9.4_

  - [ ] 10.2 Add structured logging and events
    - Implement structured logging for all operations
    - Add Kubernetes event emission for lifecycle transitions
    - Create detailed error reporting in status conditions
    - _Requirements: 7.6, 9.2, 9.3_

  - [ ]* 10.3 Write property test for observability
    - **Property 18: Observability and Events**
    - **Validates: Requirements 7.6, 7.7, 9.2, 9.4**

  - [ ] 10.4 Implement configuration management
    - Add ConfigMap and environment variable support
    - Implement external secrets integration (ExternalSecrets)
    - Add provider selection configuration
    - _Requirements: 9.1, 9.5, 10.3_

  - [ ]* 10.5 Write property test for configuration
    - **Property 22: Configuration Management**
    - **Validates: Requirements 9.1**

  - [ ]* 10.6 Write property test for external secrets
    - **Property 23: External Secret Integration**
    - **Validates: Requirements 9.5**

- [ ] 11. Implement Security and RBAC
  - [ ] 11.1 Create RBAC manifests
    - Define least-privilege ClusterRole and Role
    - Create ServiceAccount and RoleBindings
    - Add security context for rootless containers
    - _Requirements: 7.4, 7.8_

  - [ ]* 11.2 Write property test for RBAC and security
    - **Property 17: RBAC and Security Context**
    - **Validates: Requirements 7.4, 7.8**

  - [ ] 11.3 Add reconciliation behavior validation
    - Ensure all controllers properly reconcile to desired state
    - Add conflict resolution and retry logic
    - Implement proper error handling and recovery
    - _Requirements: 7.3_

  - [ ]* 11.4 Write property test for reconciliation
    - **Property 16: Reconciliation Behavior**
    - **Validates: Requirements 7.3**

- [ ] 12. Create Deployment Manifests and Documentation
  - [ ] 12.1 Create Kustomize manifests
    - Generate base manifests for all CRDs
    - Create operator deployment with proper security contexts
    - Add ConfigMap templates for configuration
    - _Requirements: 7.5_

  - [ ] 12.2 Create Helm chart
    - Package operator as Helm chart with configurable values
    - Add templates for all Kubernetes resources
    - Include RBAC and security configurations
    - _Requirements: 7.5_

  - [ ] 12.3 Add integration tests
    - Create end-to-end test scenarios
    - Test complete VM lifecycle from MachineClaim to GitHub runner
    - Add chaos testing for error scenarios
    - _Requirements: All requirements integration_

- [ ] 13. Final Integration and Testing
  - [ ] 13.1 Integration testing with real Proxmox
    - Test operator against real Proxmox VE environment
    - Validate VM provisioning and cleanup
    - Test SPIFFE integration and credential flow
    - _Requirements: All Proxmox requirements_

  - [ ] 13.2 End-to-end GitHub Actions testing
    - Test complete workflow from webhook to job execution
    - Validate runner registration and job completion
    - Test scaling behavior under load
    - _Requirements: All GitHub Actions requirements_

  - [ ] 13.3 Performance and reliability testing
    - Load test with multiple concurrent VMs
    - Test operator recovery after restarts
    - Validate resource cleanup under various failure scenarios
    - _Requirements: Performance and reliability aspects_

- [ ] 14. Final Checkpoint - Complete System
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using gopter
- Unit tests validate specific examples and edge cases
- The implementation uses Go with Operator SDK and controller-runtime framework
- Bootstrap service binary will be packaged into VM templates for deployment