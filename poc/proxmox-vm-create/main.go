package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"

	"github.com/Telmate/proxmox-api-go/proxmox"
)

func main() {
	// Get configuration from environment variables
	proxmoxURL := os.Getenv("PROXMOX_URL")     // e.g., "https://pve.example.com:8006"
	username := os.Getenv("PROXMOX_USERNAME")  // e.g., "root@pam" or "hyperfleet@pve"
	password := os.Getenv("PROXMOX_PASSWORD")  // password or API token secret
	node := os.Getenv("PROXMOX_NODE")          // e.g., "pve-node-1"

	if proxmoxURL == "" || username == "" || password == "" || node == "" {
		log.Fatal("Please set PROXMOX_URL, PROXMOX_USERNAME, PROXMOX_PASSWORD, and PROXMOX_NODE environment variables")
	}

	// Create Proxmox client - the library expects the full API URL
	apiURL := proxmoxURL + "/api2/json"
	client, err := proxmox.NewClient(apiURL, nil, "", &tls.Config{InsecureSkipVerify: true}, "", 300)
	if err != nil {
		log.Fatalf("Failed to create Proxmox client: %v", err)
	}

	// Login to Proxmox
	ctx := context.Background()
	err = client.Login(ctx, username, password, "")
	if err != nil {
		log.Fatalf("Failed to login to Proxmox: %v", err)
	}
	fmt.Println("Successfully logged in to Proxmox")

	// Helper functions for pointer values
	stringPtr := func(s string) *string { return &s }

	// Define VM configuration using the simplest approach
	vmConfig := proxmox.ConfigQemu{
		Name:   (*proxmox.GuestName)(stringPtr("test-vm")),
		QemuOs: "l26", // Linux 2.6+ kernel
		Pool:   (*proxmox.PoolName)(stringPtr("hyperfleet")),
		
		// Set VM ID and node
		ID:   (*proxmox.GuestID)(func() *proxmox.GuestID { id := proxmox.GuestID(7099); return &id }()),
		Node: (*proxmox.NodeName)(stringPtr(node)),
		
		// Memory configuration
		Memory: &proxmox.QemuMemory{
			CapacityMiB: (*proxmox.QemuMemoryCapacity)(func() *proxmox.QemuMemoryCapacity { 
				capacity := proxmox.QemuMemoryCapacity(4096)
				return &capacity 
			}()),
		},
		
		// CPU configuration
		CPU: &proxmox.QemuCPU{
			Cores: (*proxmox.QemuCpuCores)(func() *proxmox.QemuCpuCores { 
				cores := proxmox.QemuCpuCores(2)
				return &cores 
			}()),
			Sockets: (*proxmox.QemuCpuSockets)(func() *proxmox.QemuCpuSockets { 
				sockets := proxmox.QemuCpuSockets(1)
				return &sockets 
			}()),
		},
		
		// SCSI controller
		Scsihw: "virtio-scsi-pci",
		
		// Network configuration
		Networks: proxmox.QemuNetworkInterfaces{
			0: proxmox.QemuNetworkInterface{
				Model:  (*proxmox.QemuNetworkModel)(stringPtr("virtio")),
				Bridge: stringPtr("vmbr0"),
			},
		},
		
		// Basic disk configuration
		Disks: &proxmox.QemuStorages{
			Scsi: &proxmox.QemuScsiDisks{
				Disk_0: &proxmox.QemuScsiStorage{
					Disk: &proxmox.QemuScsiDisk{
						Storage:         "local-lvm",
						SizeInKibibytes: proxmox.QemuDiskSize(20 * 1024 * 1024), // 20GB in KiB
						Format:          "raw",
					},
				},
			},
		},
	}

	// Create the VM
	vmid := 7099
	fmt.Printf("Creating VM %d on node %s...\n", vmid, node)
	
	// Create the VM using the config's Create method
	vmr, err := vmConfig.Create(ctx, client)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	fmt.Printf("Successfully created VM %d (%s) on node %s\n", int(vmr.VmId()), *vmConfig.Name, node)
	
	// Optional: Get VM status
	config, err := proxmox.NewConfigQemuFromApi(ctx, vmr, client)
	if err != nil {
		log.Printf("Warning: Could not retrieve VM config: %v", err)
	} else {
		fmt.Printf("VM created successfully!\n")
		if config.Name != nil {
			fmt.Printf("VM Name: %s\n", *config.Name)
		}
		if config.Pool != nil {
			fmt.Printf("VM Pool: %s\n", *config.Pool)
		}
	}
}