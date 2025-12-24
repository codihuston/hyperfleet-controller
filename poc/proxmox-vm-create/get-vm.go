package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

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

	// Get VM ID from command line argument or use default
	vmidStr := "7099" // default
	if len(os.Args) > 1 {
		vmidStr = os.Args[1]
	}

	vmid, err := strconv.Atoi(vmidStr)
	if err != nil {
		log.Fatalf("Invalid VM ID: %s", vmidStr)
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

	// Create VM reference
	vmr := proxmox.NewVmRef(proxmox.GuestID(vmid))
	vmr.SetNode(node)
	vmr.SetVmType(proxmox.GuestQemu) // Specify that this is a QEMU VM

	// Get VM configuration
	fmt.Printf("Retrieving VM %d configuration from node %s...\n", vmid, node)
	
	config, err := proxmox.NewConfigQemuFromApi(ctx, vmr, client)
	if err != nil {
		log.Fatalf("Failed to get VM config: %v", err)
	}

	// Display VM information
	fmt.Printf("\n=== VM %d Configuration ===\n", vmid)
	
	if config.Name != nil {
		fmt.Printf("Name: %s\n", *config.Name)
	}
	
	if config.Pool != nil {
		fmt.Printf("Pool: %s\n", *config.Pool)
	}
	
	if config.Memory != nil && config.Memory.CapacityMiB != nil {
		fmt.Printf("Memory: %d MB\n", *config.Memory.CapacityMiB)
	}
	
	if config.CPU != nil {
		if config.CPU.Cores != nil {
			fmt.Printf("CPU Cores: %d\n", *config.CPU.Cores)
		}
		if config.CPU.Sockets != nil {
			fmt.Printf("CPU Sockets: %d\n", *config.CPU.Sockets)
		}
	}
	
	fmt.Printf("OS Type: %s\n", config.QemuOs)
	fmt.Printf("SCSI Hardware: %s\n", config.Scsihw)
	
	// Display network configuration
	if len(config.Networks) > 0 {
		fmt.Printf("\n=== Network Configuration ===\n")
		for id, network := range config.Networks {
			fmt.Printf("Network %d:\n", id)
			if network.Model != nil {
				fmt.Printf("  Model: %s\n", *network.Model)
			}
			if network.Bridge != nil {
				fmt.Printf("  Bridge: %s\n", *network.Bridge)
			}
		}
	}
	
	// Display disk configuration
	if config.Disks != nil && config.Disks.Scsi != nil {
		fmt.Printf("\n=== Disk Configuration ===\n")
		
		if config.Disks.Scsi.Disk_0 != nil && config.Disks.Scsi.Disk_0.Disk != nil {
			disk := config.Disks.Scsi.Disk_0.Disk
			fmt.Printf("SCSI0:\n")
			fmt.Printf("  Storage: %s\n", disk.Storage)
			fmt.Printf("  Size: %d KiB\n", disk.SizeInKibibytes)
			fmt.Printf("  Format: %s\n", disk.Format)
		}
	}

	// Get VM status (running, stopped, etc.)
	fmt.Printf("\n=== VM Status ===\n")
	
	// Try to get VM status using the client's status methods
	status, err := client.GetVmState(ctx, vmr)
	if err != nil {
		fmt.Printf("Could not retrieve VM status: %v\n", err)
	} else {
		fmt.Printf("Status: %s\n", status["status"])
		if uptime, ok := status["uptime"]; ok {
			fmt.Printf("Uptime: %v seconds\n", uptime)
		}
		if cpu, ok := status["cpu"]; ok {
			fmt.Printf("CPU Usage: %.2f%%\n", cpu.(float64)*100)
		}
		if memUsed, ok := status["mem"]; ok {
			if memMax, ok := status["maxmem"]; ok {
				memUsedFloat := memUsed.(float64)
				memMaxFloat := memMax.(float64)
				fmt.Printf("Memory Usage: %.0f MB / %.0f MB (%.1f%%)\n", 
					memUsedFloat/1024/1024, memMaxFloat/1024/1024, (memUsedFloat/memMaxFloat)*100)
			}
		}
	}

	fmt.Printf("\nVM information retrieved successfully!\n")
}