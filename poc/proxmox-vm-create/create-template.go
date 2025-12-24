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

	// Get VM ID from command line argument
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run create-template.go <vm-id>")
	}

	vmidStr := os.Args[1]
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
	vmr.SetVmType(proxmox.GuestQemu)

	// First, check if VM exists and get its current status
	fmt.Printf("Checking VM %d status...\n", vmid)
	config, err := proxmox.NewConfigQemuFromApi(ctx, vmr, client)
	if err != nil {
		log.Fatalf("Failed to get VM config: %v", err)
	}

	vmName := "unknown"
	if config.Name != nil {
		vmName = string(*config.Name)
	}

	// Check VM status - it should be stopped before converting to template
	status, err := client.GetVmState(ctx, vmr)
	if err != nil {
		log.Fatalf("Failed to get VM status: %v", err)
	}

	vmStatus := status["status"].(string)
	fmt.Printf("VM %d (%s) current status: %s\n", vmid, vmName, vmStatus)

	if vmStatus != "stopped" {
		fmt.Printf("VM must be stopped before converting to template. Current status: %s\n", vmStatus)
		fmt.Println("Stopping VM...")
		
		// Stop the VM
		_, err = client.StopVm(ctx, vmr)
		if err != nil {
			log.Fatalf("Failed to stop VM: %v", err)
		}

		// Wait for VM to stop (simple polling)
		fmt.Println("Waiting for VM to stop...")
		for i := 0; i < 30; i++ { // Wait up to 30 seconds
			status, err := client.GetVmState(ctx, vmr)
			if err != nil {
				log.Fatalf("Failed to check VM status: %v", err)
			}
			
			if status["status"].(string) == "stopped" {
				fmt.Println("VM stopped successfully")
				break
			}
			
			if i == 29 {
				log.Fatal("Timeout waiting for VM to stop")
			}
			
			fmt.Print(".")
			// Sleep for 1 second (using a simple approach)
			select {
			case <-ctx.Done():
				return
			default:
				// Continue
			}
		}
	}

	// Convert VM to template
	fmt.Printf("Converting VM %d (%s) to template...\n", vmid, vmName)
	
	// Use the client's template conversion method
	err = client.CreateTemplate(ctx, vmr)
	if err != nil {
		log.Fatalf("Failed to convert VM to template: %v", err)
	}

	fmt.Printf("Successfully converted VM %d (%s) to template!\n", vmid, vmName)
	fmt.Printf("Template ID: %d\n", vmid)
	fmt.Printf("Template Name: %s\n", vmName)
	fmt.Println("\nYou can now use this template to create new VMs with:")
	fmt.Printf("go run clone-from-template.go %d <new-vm-id>\n", vmid)
	
	// Verify template creation by checking the config
	templateConfig, err := proxmox.NewConfigQemuFromApi(ctx, vmr, client)
	if err != nil {
		log.Printf("Warning: Could not verify template creation: %v", err)
	} else {
		fmt.Printf("\nTemplate verification successful!\n")
		if templateConfig.Name != nil {
			fmt.Printf("Template Name: %s\n", *templateConfig.Name)
		}
		if templateConfig.Pool != nil {
			fmt.Printf("Template Pool: %s\n", *templateConfig.Pool)
		}
	}
}