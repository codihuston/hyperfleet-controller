package provider

import (
	"crypto/tls"
	"testing"
)

func TestDefaultClientFactory_CreateClient(t *testing.T) {
	factory := NewClientFactory()

	config := &ClientConfig{
		Endpoint:  "https://pve.example.com:8006/api2/json",
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
		Timeout:   300,
	}

	auth := &AuthConfig{
		Type:        "token",
		TokenID:     "test-token-id",
		TokenSecret: "test-token-secret",
	}

	tests := []struct {
		name        string
		provider    string
		expectError bool
		expectType  string
	}{
		{
			name:        "proxmox provider",
			provider:    "proxmox",
			expectError: false,
			expectType:  "*provider.ProxmoxClient",
		},
		{
			name:        "proxmox provider case insensitive",
			provider:    "PROXMOX",
			expectError: false,
			expectType:  "*provider.ProxmoxClient",
		},
		{
			name:        "unsupported provider",
			provider:    "vmware",
			expectError: true,
			expectType:  "",
		},
		{
			name:        "empty provider",
			provider:    "",
			expectError: true,
			expectType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := factory.CreateClient(tt.provider, config, auth)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if client != nil {
					t.Errorf("expected nil client but got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("expected client but got nil")
				}

				// Verify the client type
				if tt.expectType != "" {
					clientType := getClientType(client)
					if clientType != tt.expectType {
						t.Errorf("expected client type %s but got %s", tt.expectType, clientType)
					}
				}

				// Clean up
				if client != nil {
					if closeErr := client.Close(); closeErr != nil {
						t.Errorf("Failed to close client: %v", closeErr)
					}
				}
			}
		})
	}
}

func TestNewClientFactory(t *testing.T) {
	factory := NewClientFactory()
	if factory == nil {
		t.Errorf("expected factory but got nil")
	}

	// Verify it implements the interface
	_ = factory
}

// Helper function to get client type for testing
func getClientType(client HypervisorClient) string {
	switch client.(type) {
	case *ProxmoxClient:
		return "*provider.ProxmoxClient"
	default:
		return "unknown"
	}
}
