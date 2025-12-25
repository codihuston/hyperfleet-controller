# HyperFleet Operator Project Structure

## Overview

The HyperFleet operator follows the standard Operator SDK project layout with additional organization for hypervisor providers and observability components.

## Current Project Layout

```
hyperfleet-operator/
├── .devcontainer/                    # Development container configuration
│   ├── devcontainer.json            # VS Code dev container settings
│   └── post-install.sh              # Container setup script
├── .github/                          # GitHub workflows and templates
│   └── workflows/
│       ├── ci.yml                    # Continuous integration
│       ├── lint.yml                  # Linting workflow
│       ├── test-e2e.yml              # End-to-end testing
│       └── test.yml                  # Unit testing
├── .kiro/                            # Kiro specifications and steering
│   ├── specs/
│   │   └── hyperfleet-operator/
│   │       ├── requirements.md       # Feature requirements
│   │       ├── design.md             # Design document
│   │       └── tasks.md              # Implementation tasks
│   └── steering/
│       ├── go-best-practices.md      # Go coding standards
│       ├── development-workflow.md   # Git and development workflow
│       ├── operator-architecture.md  # Architecture decisions
│       └── project-structure.md      # This document
├── api/                              # API definitions (CRDs)
│   └── v1alpha1/
│       ├── groupversion_info.go      # API group version info
│       ├── hypervisorcluster_types.go
│       ├── hypervisormachinetemplate_types.go
│       ├── machineclaim_types.go
│       ├── runnerpool_types.go
│       └── zz_generated.deepcopy.go  # Generated deepcopy methods
├── bin/                              # Built binaries and tools
│   ├── k8s/                          # Kubernetes test binaries
│   ├── controller-gen                # Code generation tool
│   ├── setup-envtest                 # Test environment setup
│   └── manager                       # Operator binary
├── cmd/                              # Main applications
│   └── main.go                       # Operator entrypoint
├── config/                           # Kubernetes manifests
│   ├── crd/                          # Custom Resource Definitions
│   │   ├── bases/                    # Generated CRD manifests
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   ├── default/                      # Default deployment configuration
│   │   ├── cert_metrics_manager_patch.yaml
│   │   ├── kustomization.yaml
│   │   ├── manager_metrics_patch.yaml
│   │   └── metrics_service.yaml
│   ├── manager/                      # Manager deployment
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── manifests/                    # OLM bundle manifests
│   │   └── kustomization.yaml
│   ├── network-policy/               # Network security policies
│   │   ├── allow-metrics-traffic.yaml
│   │   └── kustomization.yaml
│   ├── prometheus/                   # Prometheus monitoring
│   │   ├── kustomization.yaml
│   │   ├── monitor_tls_patch.yaml
│   │   └── monitor.yaml
│   ├── rbac/                         # RBAC permissions
│   │   ├── *_admin_role.yaml         # Admin roles for each CRD
│   │   ├── *_editor_role.yaml        # Editor roles for each CRD
│   │   ├── *_viewer_role.yaml        # Viewer roles for each CRD
│   │   ├── leader_election_role*.yaml
│   │   ├── metrics_auth_role*.yaml
│   │   ├── role.yaml
│   │   ├── role_binding.yaml
│   │   ├── service_account.yaml
│   │   └── kustomization.yaml
│   ├── samples/                      # Sample custom resources
│   │   ├── hypervisor_v1alpha1_*.yaml
│   │   └── kustomization.yaml
│   └── scorecard/                    # Operator scorecard tests
│       ├── bases/
│       ├── patches/
│       └── kustomization.yaml
├── internal/                         # Private application code
│   └── controller/                   # Controllers (scaffolded)
│       ├── hypervisorcluster_controller.go
│       ├── hypervisorcluster_controller_test.go
│       ├── hypervisormachinetemplate_controller.go
│       ├── hypervisormachinetemplate_controller_test.go
│       ├── machineclaim_controller.go
│       ├── machineclaim_controller_test.go
│       ├── runnerpool_controller.go
│       ├── runnerpool_controller_test.go
│       └── suite_test.go
├── test/                             # Test files
│   ├── e2e/                          # End-to-end tests
│   │   ├── e2e_suite_test.go
│   │   └── e2e_test.go
│   └── utils/                        # Test utilities
│       └── utils.go
├── hack/                             # Build and development scripts
│   └── boilerplate.go.txt           # License header template
├── tmp/                              # Temporary files (gitignored)
├── .dockerignore                     # Docker ignore rules
├── .gitignore                        # Git ignore rules
├── .golangci.yml                     # Linter configuration
├── cover.out                         # Test coverage output
├── Dockerfile                        # Container image definition
├── go.mod                            # Go module definition
├── go.sum                            # Go module checksums
├── Makefile                          # Build automation
├── PROJECT                           # Kubebuilder project metadata
└── README.md                         # Project documentation
```

## Directory Organization Rationale

### Standard Operator SDK Layout
- **`api/v1alpha1/`**: Custom Resource Definitions following Kubernetes API conventions
- **`cmd/`**: Application entrypoints (main.go)
- **`config/`**: Kubernetes manifests organized by function (CRDs, RBAC, deployment)
- **`internal/`**: Private application code not intended for external use
- **`test/`**: All test files organized by test type

### Generated Files (Do Not Edit)
- **`api/v1alpha1/zz_generated.deepcopy.go`**: Generated by controller-gen
- **`config/crd/bases/`**: Generated CRD manifests
- **`bin/`**: Built binaries and downloaded tools
- **`cover.out`**: Test coverage reports

### Development and CI/CD
- **`.github/workflows/`**: GitHub Actions for CI/CD automation
- **`.devcontainer/`**: VS Code development container configuration
- **`.kiro/`**: Kiro specifications and steering documentation
- **`hack/`**: Build scripts and code generation utilities

### Configuration Files
- **`.golangci.yml`**: Comprehensive Go linting configuration
- **`Dockerfile`**: Multi-stage container build
- **`Makefile`**: Standard Operator SDK build targets
- **`PROJECT`**: Kubebuilder metadata for scaffolding

## Next Steps for Implementation

Based on our requirements and design, we still need to add:

### Provider Interface (`internal/provider/`)
- **`interface.go`**: Hypervisor provider interface definition
- **`proxmox/`**: Proxmox VE provider implementation
- **`mock/`**: Mock provider for testing

### Bootstrap Service (`internal/bootstrap/`)
- **`service.go`**: Bootstrap service implementation
- **`attestation/`**: Pluggable attestation methods
- **`credentials/`**: Credential management

### Observability (`internal/observability/`)
- **`logger/`**: Structured logging implementation
- **`tracing/`**: OpenTelemetry tracing setup
- **`metrics/`**: Custom Prometheus metrics

### Utility Functions (`internal/util/`)
- **`builders/`**: Resource builder pattern implementations
- **`conditions/`**: Kubernetes condition helpers
- **`finalizers/`**: Finalizer management utilities

This structure provides a solid foundation following Operator SDK conventions while providing clear organization for HyperFleet's specific architectural components. See `operator-architecture.md` for detailed architectural decisions and patterns.

## Directory Organization Rationale

### Standard Operator SDK Layout
- **`api/v1alpha1/`**: Custom Resource Definitions following Kubernetes API conventions
- **`cmd/`**: Application entrypoints (main.go)
- **`config/`**: Kubernetes manifests organized by function (CRDs, RBAC, deployment)
- **`internal/`**: Private application code not intended for external use
- **`pkg/`**: Public library code that could be imported by other projects
- **`test/`**: All test files organized by test type

### HyperFleet-Specific Organization
- **`internal/provider/`**: Hypervisor abstraction layer with pluggable providers
- **`internal/bootstrap/`**: VM bootstrap and attestation services
- **`internal/observability/`**: Centralized logging, tracing, and metrics
- **`internal/util/builders/`**: Builder pattern implementations for resource construction

### Generated and Tooling Files
- **`hack/`**: Build scripts and code generation utilities
- **`.github/workflows/`**: CI/CD automation
- **`docs/`**: Project documentation organized by audience

This structure follows Operator SDK conventions while providing clear organization for HyperFleet's specific architectural components. See `operator-architecture.md` for detailed architectural decisions and patterns.  