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

	// List VMs on the specified node
	fmt.Printf("Listing VMs on node %s...\n\n", node)
	
	vms, err := client.GetVmList()
	if err != nil {
		log.Fatalf("Failed to get VM list: %v", err)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found on this node.")
		return
	}

	fmt.Printf("%-6s %-20s %-10s %-8s %-10s %-15s\n", "VMID", "Name", "Status", "CPU%", "Memory", "Pool")
	fmt.Println("--------------------------------------------------------------------------------")

	for vmid, vmInfo := range vms {
		// Get detailed VM status
		vmr := proxmox.NewVmRef(proxmox.GuestID(vmid))
		vmr.SetNode(node)
		vmr.SetVmType(proxmox.GuestQemu)
		
		status, err := client.GetVmState(ctx, vmr)
		if err != nil {
			// Skip VMs we can't get status for (might be on different nodes)
			continue
		}

		// Extract information
		name := vmInfo["name"]
		if name == nil {
			name = "N/A"
		}
		
		vmStatus := status["status"]
		if vmStatus == nil {
			vmStatus = "unknown"
		}

		cpu := "0.00%"
		if cpuVal, ok := status["cpu"]; ok && cpuVal != nil {
			cpu = fmt.Sprintf("%.2f%%", cpuVal.(float64)*100)
		}

		memory := "N/A"
		if memUsed, ok := status["mem"]; ok && memUsed != nil {
			if memMax, ok := status["maxmem"]; ok && memMax != nil {
				memUsedMB := memUsed.(float64) / 1024 / 1024
				memMaxMB := memMax.(float64) / 1024 / 1024
				memory = fmt.Sprintf("%.0f/%.0fMB", memUsedMB, memMaxMB)
			}
		}

		pool := "N/A"
		if poolVal, ok := vmInfo["pool"]; ok && poolVal != nil {
			pool = poolVal.(string)
		}

		fmt.Printf("%-6d %-20s %-10s %-8s %-10s %-15s\n", 
			vmid, name, vmStatus, cpu, memory, pool)
	}

	fmt.Printf("\nFound %d VMs on node %s\n", len(vms), node)
}