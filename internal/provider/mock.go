package provider

import (
	"context"
)

// MockHypervisorClient implements HypervisorClient for testing
type MockHypervisorClient struct {
	TestConnectionFunc func(ctx context.Context) (*ConnectionInfo, error)
	CloseFunc          func() error
	Closed             bool
}

// TestConnection implements HypervisorClient
func (m *MockHypervisorClient) TestConnection(ctx context.Context) (*ConnectionInfo, error) {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return &ConnectionInfo{
		Version: "mock-1.0.0",
		Metadata: map[string]string{
			"provider": "mock",
			"type":     "test",
		},
	}, nil
}

// Close implements HypervisorClient
func (m *MockHypervisorClient) Close() error {
	m.Closed = true
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockClientFactory implements ClientFactory for testing
type MockClientFactory struct {
	CreateClientFunc func(provider string, config *ClientConfig, auth *AuthConfig) (HypervisorClient, error)
	CreatedClients   []HypervisorClient
}

// CreateClient implements ClientFactory
func (m *MockClientFactory) CreateClient(provider string, config *ClientConfig, auth *AuthConfig) (HypervisorClient, error) {
	if m.CreateClientFunc != nil {
		client, err := m.CreateClientFunc(provider, config, auth)
		if err == nil && client != nil {
			m.CreatedClients = append(m.CreatedClients, client)
		}
		return client, err
	}

	// Default mock behavior
	client := &MockHypervisorClient{}
	m.CreatedClients = append(m.CreatedClients, client)
	return client, nil
}

// NewMockClientFactory creates a new mock client factory
func NewMockClientFactory() *MockClientFactory {
	return &MockClientFactory{
		CreatedClients: make([]HypervisorClient, 0),
	}
}

// NewFailingMockClientFactory creates a mock factory that always fails
func NewFailingMockClientFactory(err error) *MockClientFactory {
	return &MockClientFactory{
		CreateClientFunc: func(provider string, config *ClientConfig, auth *AuthConfig) (HypervisorClient, error) {
			return nil, err
		},
		CreatedClients: make([]HypervisorClient, 0),
	}
}

// NewMockClientFactoryWithClient creates a mock factory that returns a specific client
func NewMockClientFactoryWithClient(client HypervisorClient) *MockClientFactory {
	return &MockClientFactory{
		CreateClientFunc: func(provider string, config *ClientConfig, auth *AuthConfig) (HypervisorClient, error) {
			return client, nil
		},
		CreatedClients: []HypervisorClient{client},
	}
}
