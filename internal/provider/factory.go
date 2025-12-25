package provider

import (
	"fmt"
	"strings"
)

// DefaultClientFactory implements ClientFactory
type DefaultClientFactory struct{}

// NewClientFactory creates a new client factory
func NewClientFactory() ClientFactory {
	return &DefaultClientFactory{}
}

// CreateClient creates a hypervisor client based on provider type
func (f *DefaultClientFactory) CreateClient(provider string, config *ClientConfig, auth *AuthConfig) (HypervisorClient, error) {
	switch strings.ToLower(provider) {
	case "proxmox":
		return NewProxmoxClient(config, auth)
	default:
		return nil, fmt.Errorf("unsupported hypervisor provider: %s", provider)
	}
}
