# Operator Architecture and Deployment Guidelines

## Local Development Architecture

### Development Environment Requirements
- **Everything must be runnable locally**
- **Everything must be testable locally**
- Support for individual component testing
- Full operator testing in local clusters

### Operator SDK Integration
- Use operator-sdk scaffolding for consistent structure
- Support individual Custom Resource deployment
- Enable full operator deployment with all operands
- Maintain flexibility for different deployment scenarios

## Architectural Design Decisions

### API Structure (`api/v1alpha1/`)
- **Single API version**: Start with v1alpha1, plan for graduation to v1beta1 and v1
- **Resource separation**: Each custom resource in its own file for maintainability
- **Generated code**: Kubebuilder generates deepcopy methods automatically
- **Kubernetes conventions**: Follow standard API group/version/kind patterns

### Controller Organization (`internal/controller/`)
- **One controller per resource**: Following controller-runtime best practices
- **Shared test suite**: Common test setup in `suite_test.go`
- **Reconciliation focus**: Each controller handles its specific resource lifecycle
- **Event-driven architecture**: Controllers respond to resource changes via reconciliation loops

### Provider Interface (`internal/provider/`)
- **Pluggable architecture**: Interface-based design for multiple hypervisors
- **Proxmox implementation**: Initial provider for Proxmox VE
- **Mock provider**: For testing without real hypervisor infrastructure
- **Future extensibility**: Easy to add VMware, Hyper-V, etc.
- **Dependency injection**: Providers injected into controllers for testability

### Bootstrap Service (`internal/bootstrap/`)
- **Attestation abstraction**: Pluggable attestation methods
- **Credential management**: Secure handling of short-lived credentials
- **Join token default**: Simple attestation method for initial implementation
- **TPM support**: Hardware-based attestation for enhanced security
- **Service isolation**: Bootstrap runs as separate service component

### Builder Pattern (`internal/util/builders/`)
- **Fluent interfaces**: Easy resource construction for testing
- **Server-side apply**: Integration with Kubernetes server-side apply
- **Type safety**: Compile-time validation of resource construction
- **Immutable objects**: Builders create immutable resource instances

### Custom Resource Design Patterns

### Builder Pattern Implementation
- Use builder pattern for all custom resource instantiation
- Provide fluent interfaces for resource configuration
- Enable easy testing through builder flexibility

```go
// Example: Fleet resource builder
fleet := NewFleetBuilder("my-fleet", "default").
    WithReplicas(3).
    WithImage("nginx:latest").
    WithSelector(map[string]string{"app": "nginx"}).
    Build()
```

### Server-Side Apply Strategy
- Leverage server-side apply for all resource operations
- Use appropriate field managers for different components
- Handle ownership and conflict resolution properly

```go
// Example: Server-side apply usage
func (r *FleetReconciler) applyFleet(ctx context.Context, fleet *hyperfleetv1.Fleet) error {
    return r.Patch(ctx, fleet, client.Apply, &client.PatchOptions{
        FieldManager: "hyperfleet-controller",
        Force:        &[]bool{true}[0],
    })
}
```

## Testing Architecture

### Testing Strategy Overview
- **Unit tests**: Co-located with source code (`*_test.go`)
- **Integration tests**: Controller behavior with real Kubernetes API
- **End-to-end tests**: Full operator functionality in test clusters
- **Test data**: Fixtures and sample resources for consistent testing

### Unit Testing Strategy
- Test all business logic in isolation
- Use dependency injection for testability
- Mock external dependencies appropriately
- Focus on behavior verification, not implementation

### Integration Testing
- Test controller reconciliation logic
- Use envtest for Kubernetes API integration
- Test custom resource lifecycle management
- Verify operator behavior in realistic scenarios

### Local Cluster Testing
- Support kind/minikube for local testing
- Test full operator deployment scenarios
- Validate custom resource behavior end-to-end
- Test upgrade and migration scenarios

## Deployment Flexibility

### Individual Component Deployment
- Support deploying single custom resources
- Enable testing of specific CRDs in isolation
- Provide examples for manual resource creation

### Full Operator Deployment
- Deploy complete operator with all operands
- Support different configuration scenarios
- Enable easy local development workflows

### Configuration Management
- Use ConfigMaps for operator configuration
- Support environment-specific settings
- Enable feature flags for development/testing

## Monitoring and Observability

### Centralized Logging Strategy
- **Use structured logging** with consistent field names and formats
- **Support configurable log levels** via LOG_LEVEL environment variable (DEBUG, INFO, WARN, ERROR)
- **Default to INFO level** when LOG_LEVEL is not specified
- **Include relevant context** in all log messages (request IDs, resource names, namespaces)
- **Log major operations** including reconciliation loops, API calls, and state transitions
- **Use appropriate log levels**: DEBUG for detailed troubleshooting, INFO for normal operations, WARN for recoverable issues, ERROR for failures

```go
// Example: Structured logging with context
logger := log.WithFields(log.Fields{
    "controller": "fleet",
    "namespace":  req.Namespace,
    "name":       req.Name,
    "traceID":    span.SpanContext().TraceID(),
})
logger.Info("Starting fleet reconciliation")
```

### OpenTelemetry Tracing
- **Implement distributed tracing** using OpenTelemetry Go SDK
- **Create spans for major operations**: reconciliation loops, API calls, VM operations
- **Include relevant attributes** in spans (resource types, operation names, error states)
- **Support configurable exporters** via environment variables
- **Correlate traces with logs** using trace IDs in log messages

```go
// Example: OpenTelemetry span creation
ctx, span := tracer.Start(ctx, "fleet.reconcile",
    trace.WithAttributes(
        attribute.String("fleet.name", fleet.Name),
        attribute.String("fleet.namespace", fleet.Namespace),
    ))
defer span.End()
```

### Metrics Security (CRITICAL - Kubebuilder v4.1.0+ Required)
- **Use Controller-Runtime built-in auth**: `WithAuthenticationAndAuthorization` for metrics protection
- **Avoid deprecated kube-rbac-proxy**: gcr.io/kubebuilder images will be unavailable from March 2025
- **Enable secure metrics by default**: Use integrated authn/authz mechanisms
- **TLS encryption**: Support cert-manager integration for production deployments

```go
// Example: Secure metrics configuration (Kubebuilder v4.1.0+)
metricsServerOptions := metricsserver.Options{
    BindAddress:   metricsAddr,
    SecureServing: true, // Enable secure metrics
    TLSOpts:       tlsOpts,
}

if secureMetrics {
    // Use Controller-Runtime's built-in auth protection
    metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
}
```

### Environment Variable Configuration
- **LOG_LEVEL**: Set logging level (DEBUG, INFO, WARN, ERROR)
- **OTEL_EXPORTER_OTLP_ENDPOINT**: OpenTelemetry collector endpoint
- **OTEL_SERVICE_NAME**: Service name for tracing (default: "hyperfleet-operator")
- **OTEL_RESOURCE_ATTRIBUTES**: Additional resource attributes for traces
- **METRICS_BIND_ADDRESS**: Prometheus metrics endpoint (default: ":8080")
- **HEALTH_PROBE_BIND_ADDRESS**: Health check endpoint (default: ":8081")

### ConfigMaps and Secrets
- **Hypervisor credentials**: Stored in Kubernetes secrets with proper RBAC
- **Operator configuration**: Non-sensitive config via ConfigMaps
- **External secret management**: Integration with ExternalSecrets operator
- **Feature flags**: Runtime configuration for development/testing scenarios

### Metrics and Health Checks
- **Prometheus metrics**: Custom metrics for VM provisioning, API latency, error rates
- **Health endpoints**: Readiness and liveness probes for Kubernetes deployment
- **Performance monitoring**: Track reconciliation loop performance and resource utilization
- **Alerting integration**: Metrics designed for alerting on operational issues

## Development Workflow Integration

### Project Scaffolding Requirements (CRITICAL)
- **Use Kubebuilder v4.1.0 or later**: Required to avoid deprecated gcr.io/kubebuilder images
- **Enable secure metrics by default**: Use Controller-Runtime's built-in authentication/authorization
- **Avoid kube-rbac-proxy**: The gcr.io/kubebuilder/kube-rbac-proxy images will be unavailable from March 2025
- **Verify scaffolding version**: Ensure generated code uses `filters.WithAuthenticationAndAuthorization`

```bash
# Verify Kubebuilder version before scaffolding
kubebuilder version
# Should be v4.1.0 or later

# Initialize project with correct settings
operator-sdk init --domain=hyperfleet.io --repo=github.com/your-org/hyperfleet-operator
```

### Local Development Workflow
1. **Start local cluster**: `kind create cluster` or `minikube start`
2. **Install CRDs**: `make install`
3. **Run operator locally**: `make run`
4. **Apply sample resources**: `kubectl apply -f config/samples/`
5. **Test individual components**: Deploy single CRDs for focused testing

### Testing Workflow
1. **Unit tests**: `make test` - Fast feedback on business logic
2. **Integration tests**: `make test-integration` - Controller behavior validation
3. **End-to-end tests**: `make test-e2e` - Full operator functionality
4. **Linting**: `make lint` - Code quality and style enforcement

### Build and Deployment Workflow
1. **Build binary**: `make build` - Local development binary
2. **Build container**: `make docker-build` - Containerized operator
3. **Deploy to cluster**: `make deploy` - Full operator deployment
4. **Generate manifests**: `make manifests` - Update CRDs and RBAC

## Development Tools Integration

### Operator SDK Usage
- Follow operator-sdk best practices
- Use generated scaffolding as foundation
- Customize as needed for specific requirements

### Local Development Workflow
1. Start local Kubernetes cluster (kind/minikube)
2. Deploy CRDs individually for testing
3. Run operator locally against cluster
4. Test custom resource lifecycle
5. Validate behavior and performance