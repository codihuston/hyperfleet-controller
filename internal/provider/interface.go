package provider

import (
	"context"
	"crypto/tls"
)

// HypervisorClient defines the interface for hypervisor client adapters
type HypervisorClient interface {
	// TestConnection validates the connection to the hypervisor
	TestConnection(ctx context.Context) (*ConnectionInfo, error)

	// Close cleans up any resources used by the client
	Close() error
}

// ConnectionInfo contains information about a successful hypervisor connection
type ConnectionInfo struct {
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ClientConfig contains common configuration for hypervisor clients
type ClientConfig struct {
	Endpoint  string
	TLSConfig *tls.Config
	Timeout   int // timeout in seconds
}

// AuthConfig contains authentication information
type AuthConfig struct {
	Type        string // "token", "password", etc.
	TokenID     string
	TokenSecret string
	Username    string
	Password    string
}

// ClientFactory creates hypervisor clients based on provider type
type ClientFactory interface {
	CreateClient(provider string, config *ClientConfig, auth *AuthConfig) (HypervisorClient, error)
}
