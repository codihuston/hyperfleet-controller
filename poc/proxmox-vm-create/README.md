# Proxmox VM Creation Example

This example demonstrates how to create a VM in Proxmox VE using the Go API client.

## Prerequisites

1. Proxmox VE server with API access
2. Valid credentials (username/password or API token)
3. Go 1.21 or later

## Setup

1. Initialize and install dependencies:
```bash
cd examples/proxmox-vm-create
go mod init proxmox-vm-create
go get github.com/Telmate/proxmox-api-go/proxmox
go mod tidy
```

2. Set environment variables:
```bash
# For username/password authentication
export PROXMOX_URL="https://your-proxmox-server:8006"
export PROXMOX_USERNAME="root@pam"
export PROXMOX_PASSWORD="your-password"
export PROXMOX_NODE="your-node-name"

# For API token authentication
export PROXMOX_URL="https://your-proxmox-server:8006"
export PROXMOX_USERNAME="hyperfleet@pve!mytoken"
export PROXMOX_PASSWORD="your-token-secret"
export PROXMOX_NODE="your-node-name"
```

## Usage

### Creating a VM

Run the example with your Proxmox credentials:
```bash
export PROXMOX_URL='https://your-proxmox-server:8006'
export PROXMOX_USERNAME='root@pam'
export PROXMOX_PASSWORD='your-password'
export PROXMOX_NODE='your-node-name'
cd examples/proxmox-vm-create
go run main.go
```

### Getting VM Information

To retrieve information about an existing VM:
```bash
# Get info for VM ID 7099 (default)
go run get-vm.go

# Get info for a specific VM ID
go run get-vm.go 1001
```

The get-vm script displays:
- VM configuration (name, pool, memory, CPU)
- Network configuration
- Disk configuration  
- Current VM status and resource usage

### Listing All VMs

To list all VMs on the node:
```bash
go run list-vms.go
```

### Creating Templates and Cloning VMs

#### Convert VM to Template

To convert an existing VM to a template (VM must be stopped):
```bash
# Convert VM 7099 to a template
go run create-template.go 7099
```

This will:
- Stop the VM if it's running
- Convert it to a template
- Verify the template creation

#### Clone VM from Template

To create a new VM from a template:
```bash
# Clone template 7099 to new VM 8001 with name "my-runner"
go run clone-from-template.go 7099 8001 "my-runner"

# Clone with auto-generated name
go run clone-from-template.go 7099 8002
```

This creates a **linked clone** by default (faster, uses less storage). For a full independent clone, modify the script to use `"full": 1` in the clone parameters.

This will create a VM with the following specifications:
- **VM ID**: 7099
- **Name**: test-vm
- **Memory**: 4096 MB
- **CPU**: 2 cores, 1 socket
- **OS Type**: Linux 2.6+ kernel
- **SCSI Controller**: virtio-scsi-pci
- **Pool**: hyperfleet
- **Disk**: 20GB on local-lvm storage
- **Network**: virtio on vmbr0 bridge

## Configuration Notes

### Storage
The example uses `local-lvm` storage. Adjust this to match your Proxmox storage configuration:
- `local-lvm` - LVM storage
- `local` - Directory storage
- `ceph` - Ceph storage
- etc.

### Network Bridge
The example uses `vmbr0` bridge. Adjust to match your network configuration.

### VM Template vs New VM
This example creates a new VM from scratch. To clone from a template instead:

```go
// Clone from template instead of creating new VM
templateVmr := pxapi.NewVmRef(9000) // Template VM ID
templateVmr.SetNode(node)

cloneParams := map[string]interface{}{
    "newid":   vmid,
    "name":    "test-vm",
    "target":  node,
    "full":    0, // Linked clone (faster)
    "storage": "local-lvm",
}

_, err = client.CloneQemuVm(templateVmr, cloneParams)
```

## Error Handling

Common issues and solutions:

1. **Authentication failed**: Check username/password or API token
2. **VM ID already exists**: Change the VM ID or delete existing VM
3. **Storage not found**: Verify storage name in Proxmox
4. **Node not found**: Verify node name in Proxmox cluster
5. **Pool not found**: Create the pool first or remove pool parameter

## API Token Setup

To create an API token in Proxmox:

1. Go to Datacenter → Permissions → API Tokens
2. Click "Add"
3. Set User: `hyperfleet@pve`
4. Set Token ID: `mytoken`
5. Uncheck "Privilege Separation" for full permissions
6. Copy the generated token secret

Use the token as:
- Username: `hyperfleet@pve!mytoken`
- Password: `<token-secret>`

## HyperFleet Workflow

This example demonstrates the core VM lifecycle that HyperFleet will implement:

1. **Template Preparation**: Create a base VM with your desired OS and software, then convert to template
2. **Fast Provisioning**: Clone new VMs from templates in seconds (linked clones)
3. **Ephemeral Usage**: VMs run CI jobs and self-terminate
4. **Cleanup**: HyperFleet automatically removes stopped VMs

### Typical HyperFleet Flow

```bash
# 1. Create and configure a base VM
go run main.go

# 2. Install software, configure the VM (done manually or via automation)
# ... VM setup process ...

# 3. Convert to template
go run create-template.go 7099

# 4. Clone ephemeral VMs as needed (this is what HyperFleet will do automatically)
go run clone-from-template.go 7099 8001 "github-runner-1"
go run clone-from-template.go 7099 8002 "github-runner-2"

# 5. VMs run jobs and terminate (handled by bootstrap service)
# 6. HyperFleet detects stopped VMs and cleans them up
```

This provides the foundation for HyperFleet's VM-per-job execution model with fast provisioning from templates.

## Available Scripts

- `main.go` - Create a new VM from scratch
- `get-vm.go` - Retrieve VM configuration and status
- `list-vms.go` - List all VMs on the node
- `create-template.go` - Convert a VM to a template
- `clone-from-template.go` - Create a new VM from a template

These scripts demonstrate the complete VM lifecycle management that HyperFleet will automate for ephemeral CI/CD workloads.