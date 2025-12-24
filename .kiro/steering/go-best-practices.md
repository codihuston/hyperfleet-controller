# Go Best Practices and Development Guidelines

## Idiomatic Go Practices

### Core Principles
- Follow effective Go practices as outlined in [Effective Go](https://go.dev/doc/effective_go)
- Write simple, readable, and maintainable code
- Embrace Go's philosophy: "Clear is better than clever"
- Use gofmt, golint, and go vet consistently

### Interface and Struct Design
- **Pass interfaces, return structs** where it makes sense
- Keep interfaces small and focused (interface segregation principle)
- Define interfaces at the point of use, not at the point of implementation
- Use concrete types for return values to avoid unnecessary abstraction

```go
// Good: Accept interface, return struct
func ProcessData(reader io.Reader) (*ProcessedData, error) {
    // Implementation
}

// Good: Small, focused interface
type DataProcessor interface {
    Process(data []byte) error
}
```

### Testing Strategy
- **Everything must be unit testable**
- Use dependency injection to enable testing
- Mock interfaces when needed, but don't overuse mocking
- Prefer table-driven tests for multiple test cases
- Test behavior, not implementation details

```go
// Good: Testable function with dependency injection
func NewService(client HTTPClient, logger Logger) *Service {
    return &Service{client: client, logger: logger}
}

// Good: Interface for mocking
type HTTPClient interface {
    Get(url string) (*http.Response, error)
}
```

## Kubernetes Operator Patterns

### Custom Resource Management
- Leverage **server-side apply** for all custom resource operations
- Use the **builder pattern** for instantiating custom resources
- Follow Kubernetes API conventions and best practices

```go
// Example: Builder pattern for custom resources
type FleetBuilder struct {
    fleet *hyperfleetv1.Fleet
}

func NewFleetBuilder(name, namespace string) *FleetBuilder {
    return &FleetBuilder{
        fleet: &hyperfleetv1.Fleet{
            ObjectMeta: metav1.ObjectMeta{
                Name:      name,
                Namespace: namespace,
            },
        },
    }
}

func (b *FleetBuilder) WithReplicas(replicas int32) *FleetBuilder {
    b.fleet.Spec.Replicas = replicas
    return b
}

func (b *FleetBuilder) Build() *hyperfleetv1.Fleet {
    return b.fleet
}
```

### Server-Side Apply Usage
- Use server-side apply for declarative resource management
- Set appropriate field managers for different components
- Handle conflicts gracefully

## Local Development Requirements

### Development Environment
- **Everything must be runnable locally**
- **Everything must be testable locally**
- Use operator-sdk scaffolding for deployment flexibility
- Support individual CR deployment and full operator deployment

### Testing Requirements
- Unit tests for all business logic
- Integration tests for controller behavior
- Local cluster testing (kind/minikube)
- Mock external dependencies appropriately

## References
- [Idiomatic Go Discussion](https://www.reddit.com/r/golang/comments/5b2j38/what_is_idiomatic_go/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Builder Pattern in Go](https://refactoring.guru/design-patterns/builder/go/example)
- [Kubernetes Server-Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)