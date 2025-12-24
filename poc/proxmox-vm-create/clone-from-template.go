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

	// Get template ID and new VM ID from command line arguments
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run clone-from-template.go <template-id> <new-vm-id> [new-vm-name]")
	}

	templateIDStr := os.Args[1]
	newVMIDStr := os.Args[2]
	
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		log.Fatalf("Invalid template ID: %s", templateIDStr)
	}

	newVMID, err := strconv.Atoi(newVMIDStr)
	if err != nil {
		log.Fatalf("Invalid new VM ID: %s", newVMIDStr)
	}

	// Optional new VM name
	newVMName := fmt.Sprintf("cloned-vm-%d", newVMID)
	if len(os.Args) > 3 {
		newVMName = os.Args[3]
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

	// Create template VM reference
	templateVMR := proxmox.NewVmRef(proxmox.GuestID(templateID))
	templateVMR.SetNode(node)
	templateVMR.SetVmType(proxmox.GuestQemu)

	// Verify template exists
	fmt.Printf("Verifying template %d exists...\n", templateID)
	templateConfig, err := proxmox.NewConfigQemuFromApi(ctx, templateVMR, client)
	if err != nil {
		log.Fatalf("Failed to get template config: %v", err)
	}

	templateName := "unknown"
	if templateConfig.Name != nil {
		templateName = string(*templateConfig.Name)
	}

	fmt.Printf("Found template %d (%s)\n", templateID, templateName)

	// Check if new VM ID already exists
	newVMR := proxmox.NewVmRef(proxmox.GuestID(newVMID))
	newVMR.SetNode(node)
	newVMR.SetVmType(proxmox.GuestQemu)

	_, err = proxmox.NewConfigQemuFromApi(ctx, newVMR, client)
	if err == nil {
		log.Fatalf("VM with ID %d already exists. Please choose a different ID.", newVMID)
	}

	// Prepare clone parameters
	fmt.Printf("Cloning template %d (%s) to new VM %d (%s)...\n", templateID, templateName, newVMID, newVMName)

	cloneParams := map[string]interface{}{
		"newid":  newVMID,
		"name":   newVMName,
		"target": node,
		"full":   0, // Linked clone (faster, uses less space)
	}

	// Perform the clone operation
	_, err = client.CloneQemuVm(ctx, templateVMR, cloneParams)
	if err != nil {
		log.Fatalf("Failed to clone VM: %v", err)
	}

	fmt.Printf("Successfully cloned template %d to VM %d!\n", templateID, newVMID)

	// Wait a moment for the clone to complete and then verify
	fmt.Println("Verifying cloned VM...")
	
	clonedConfig, err := proxmox.NewConfigQemuFromApi(ctx, newVMR, client)
	if err != nil {
		log.Printf("Warning: Could not verify cloned VM: %v", err)
	} else {
		fmt.Printf("\n=== Cloned VM %d Configuration ===\n", newVMID)
		
		if clonedConfig.Name != nil {
			fmt.Printf("Name: %s\n", *clonedConfig.Name)
		}
		
		if clonedConfig.Pool != nil {
			fmt.Printf("Pool: %s\n", *clonedConfig.Pool)
		}
		
		if clonedConfig.Memory != nil && clonedConfig.Memory.CapacityMiB != nil {
			fmt.Printf("Memory: %d MB\n", *clonedConfig.Memory.CapacityMiB)
		}
		
		if clonedConfig.CPU != nil {
			if clonedConfig.CPU.Cores != nil {
				fmt.Printf("CPU Cores: %d\n", *clonedConfig.CPU.Cores)
			}
			if clonedConfig.CPU.Sockets != nil {
				fmt.Printf("CPU Sockets: %d\n", *clonedConfig.CPU.Sockets)
			}
		}
	}

	// Get VM status
	status, err := client.GetVmState(ctx, newVMR)
	if err != nil {
		fmt.Printf("Could not get VM status: %v\n", err)
	} else {
		fmt.Printf("Status: %s\n", status["status"])
	}

	fmt.Printf("\nCloned VM is ready! You can:\n")
	fmt.Printf("- View details: go run get-vm.go %d\n", newVMID)
	fmt.Printf("- Start the VM: qm start %d (via Proxmox CLI)\n", newVMID)
	fmt.Printf("- Or start via web interface\n")
	
	fmt.Printf("\nNote: This was a linked clone, so it depends on the original template.\n")
	fmt.Printf("For a full independent clone, modify the clone parameters to use 'full': 1\n")
}