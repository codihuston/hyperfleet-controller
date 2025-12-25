package provider

import (
	"context"
	"fmt"

	"github.com/Telmate/proxmox-api-go/proxmox"
)

// ProxmoxClient implements HypervisorClient for Proxmox VE
type ProxmoxClient struct {
	client *proxmox.Client
	auth   *AuthConfig
}

// NewProxmoxClient creates a new Proxmox client adapter
func NewProxmoxClient(config *ClientConfig, auth *AuthConfig) (*ProxmoxClient, error) {
	if config == nil {
		return nil, fmt.Errorf("client config is required")
	}
	if auth == nil {
		return nil, fmt.Errorf("auth config is required")
	}

	// Create Proxmox client with TLS configuration
	client, err := proxmox.NewClient(config.Endpoint, nil, "", config.TLSConfig, "", config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	return &ProxmoxClient{
		client: client,
		auth:   auth,
	}, nil
}

// TestConnection validates the connection to Proxmox VE
func (p *ProxmoxClient) TestConnection(ctx context.Context) (*ConnectionInfo, error) {
	// Authenticate based on type
	switch p.auth.Type {
	case "token":
		// For API tokens, use SetAPIToken method
		p.client.SetAPIToken(p.auth.TokenID, p.auth.TokenSecret)
	case "password":
		// For username/password, use Login method
		err := p.client.Login(ctx, p.auth.Username, p.auth.Password, "")
		if err != nil {
			return nil, fmt.Errorf("failed to login to Proxmox: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported authentication type: %s", p.auth.Type)
	}

	// Test connection by getting version info
	version, err := p.client.GetVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Proxmox version: %w", err)
	}

	return &ConnectionInfo{
		Version: version.String(),
		Metadata: map[string]string{
			"provider": "proxmox",
			"type":     "pve",
		},
	}, nil
}

// Close cleans up any resources used by the Proxmox client
func (p *ProxmoxClient) Close() error {
	// Proxmox client doesn't require explicit cleanup
	return nil
}
