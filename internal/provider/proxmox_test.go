package provider

import (
	"context"
	"crypto/tls"
	"testing"
)

func TestNewProxmoxClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *ClientConfig
		auth        *AuthConfig
		expectError bool
	}{
		{
			name: "valid token auth config",
			config: &ClientConfig{
				Endpoint:  "https://pve.example.com:8006/api2/json",
				TLSConfig: &tls.Config{InsecureSkipVerify: true},
				Timeout:   300,
			},
			auth: &AuthConfig{
				Type:        "token",
				TokenID:     "test-token-id",
				TokenSecret: "test-token-secret",
			},
			expectError: false,
		},
		{
			name: "valid password auth config",
			config: &ClientConfig{
				Endpoint:  "https://pve.example.com:8006/api2/json",
				TLSConfig: &tls.Config{InsecureSkipVerify: true},
				Timeout:   300,
			},
			auth: &AuthConfig{
				Type:     "password",
				Username: "root@pam",
				Password: "test-password",
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			auth:        &AuthConfig{Type: "token"},
			expectError: true,
		},
		{
			name:        "nil auth",
			config:      &ClientConfig{Endpoint: "https://pve.example.com:8006/api2/json"},
			auth:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewProxmoxClient(tt.config, tt.auth)

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
			}
		})
	}
}

func TestProxmoxClient_Close(t *testing.T) {
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

	client, err := NewProxmoxClient(config, auth)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if closeErr := client.Close(); closeErr != nil {
		t.Errorf("unexpected error closing client: %v", closeErr)
	}
}

// Note: TestConnection requires a real Proxmox server, so we'll test the error cases
func TestProxmoxClient_TestConnection_InvalidAuth(t *testing.T) {
	config := &ClientConfig{
		Endpoint:  "https://pve.example.com:8006/api2/json",
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
		Timeout:   300,
	}

	tests := []struct {
		name string
		auth *AuthConfig
	}{
		{
			name: "unsupported auth type",
			auth: &AuthConfig{
				Type: "unsupported",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewProxmoxClient(config, tt.auth)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			defer func() {
				if closeErr := client.Close(); closeErr != nil {
					t.Errorf("Failed to close client: %v", closeErr)
				}
			}()

			ctx := context.Background()
			_, err = client.TestConnection(ctx)
			if err == nil {
				t.Errorf("expected error for unsupported auth type")
			}
		})
	}
}
