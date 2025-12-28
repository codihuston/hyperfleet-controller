# Pull Request

## Goal
Implement the HypervisorMachineTemplate controller to enable template validation and management for VM provisioning. This completes the core template management functionality required for the HyperFleet operator to validate and track VM templates across different hypervisor environments.

## Changes Made
- **Complete HypervisorMachineTemplate CRD implementation** with comprehensive spec including Proxmox configuration, resource requirements, attestation, bootstrap, and network settings
- **Full controller implementation** with template validation, cluster reference resolution, and status management
- **Provider factory integration** for multi-hypervisor support while maintaining extensible architecture
- **Proper finalizer handling** and cleanup logic for resource deletion
- **Periodic revalidation** with 5-minute intervals and condition-based status reporting
- **Updated design documentation** to separate attestation (VM identity) from bootstrap (workload credentials)
- **Comprehensive validation** including CPU limits, Proxmox disk format (G units), and enum validation
- **RBAC permissions** for accessing HypervisorCluster resources
- **Linting and security compliance** with proper error handling and constants

## Type of Change
- [x] ✨ New feature (non-breaking change which adds functionality)
- [x] 📚 Documentation update
- [x] 🧪 Test changes

## Impact Assessment
- [x] **API Changes**: Custom Resource Definitions or API modified
- [x] **Product Changes**: User-facing functionality modified

## Testing
- [x] Unit tests added/updated
- [x] Integration tests added/updated
- [x] Manual testing completed

## Documentation
- [x] Code comments updated
- [x] API documentation updated
- [x] Architecture documentation updated

## Checklist
- [x] Code follows the project's coding standards
- [x] Self-review of code completed
- [x] Code is properly commented, particularly in hard-to-understand areas
- [x] Corresponding changes to documentation made
- [x] Changes generate no new warnings
- [x] All tests pass locally
- [x] Lint checks pass
- [x] Conventional commit format used
- [x] PR title follows conventional commit format

## Related Issues
Implements requirements from:
- Task 2.3: Create HypervisorMachineTemplate CRD and Go types
- Task 6.3: Create HypervisorMachineTemplate controller
- Requirements 2.1, 2.2, 2.5: Template validation, reference resolution, availability monitoring

## Key Features Implemented

### HypervisorMachineTemplate CRD
- **Proxmox Integration**: Template ID, clone, and linked clone configuration
- **Resource Specifications**: CPU (1-64), memory (Kubernetes format), disk (Proxmox G format)
- **Dual Configuration**: Separate attestation (VM identity) and bootstrap (workload credentials)
- **GitHub Runner Support**: PAT and GitHub App credential options
- **Network Configuration**: DHCP, static IP, and cloud-init modes
- **Validation**: Comprehensive field validation with proper patterns and enums

### Controller Implementation
- **Template Validation**: Validates template specs against hypervisor capabilities
- **Cluster Dependency**: Ensures referenced HypervisorCluster is ready before validation
- **Provider Integration**: Uses provider factory for extensible hypervisor support
- **Status Management**: Updates conditions and availability status with detailed reasons
- **Periodic Monitoring**: Revalidates templates every 5 minutes
- **Proper Cleanup**: Finalizer-based resource cleanup

### Architecture Benefits
- **Multi-Hypervisor Ready**: Provider pattern supports future VMware, Hyper-V additions
- **Extensible Design**: Easy to add new hypervisor types through provider interface
- **Kubernetes Native**: Follows controller-runtime patterns and conventions
- **Security Compliant**: Zero security issues, proper error handling

## Deployment Notes
- Requires HypervisorCluster resources to be created first for template validation
- Controller will automatically validate templates against referenced clusters
- Templates marked as unavailable if referenced cluster is not ready

## Rollback Plan
- Remove HypervisorMachineTemplate CRD: `kubectl delete crd hypervisormachinetemplates.hypervisor.hyperfleet.io`
- Controller gracefully handles missing CRDs and will not crash
- No breaking changes to existing resources

---

**Reviewer Guidelines:**
- [x] Code quality and adherence to standards
- [x] Test coverage is adequate
- [x] Documentation is clear and complete
- [x] Security implications considered
- [x] Performance implications considered
- [x] Breaking changes properly communicated
