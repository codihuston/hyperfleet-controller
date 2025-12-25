/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HypervisorClusterSpec defines the desired state of HypervisorCluster.
type HypervisorClusterSpec struct {
	// Provider specifies the hypervisor type (e.g., "proxmox")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=proxmox
	Provider string `json:"provider"`

	// Endpoint is the API endpoint URL for the hypervisor
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://.*`
	Endpoint string `json:"endpoint"`

	// Credentials contains authentication information for the hypervisor
	// +kubebuilder:validation:Required
	Credentials HypervisorCredentials `json:"credentials"`

	// Nodes is a list of hypervisor nodes available in this cluster
	// +kubebuilder:validation:MinItems=1
	Nodes []string `json:"nodes"`

	// DefaultStorage specifies the default storage pool for VMs
	// +kubebuilder:validation:Required
	DefaultStorage string `json:"defaultStorage"`

	// DefaultNetwork specifies the default network bridge for VMs
	// +kubebuilder:validation:Required
	DefaultNetwork string `json:"defaultNetwork"`

	// DNS configuration for VMs created on this cluster
	// +optional
	DNS *DNSConfig `json:"dns,omitempty"`

	// Tags are key-value pairs applied to all VMs created on this cluster
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// HypervisorCredentials defines authentication methods for hypervisor access.
type HypervisorCredentials struct {
	// TokenID references a secret containing the API token ID (Proxmox)
	// +optional
	TokenID *corev1.SecretKeySelector `json:"tokenId,omitempty"`

	// TokenSecret references a secret containing the API token secret (Proxmox)
	// +optional
	TokenSecret *corev1.SecretKeySelector `json:"tokenSecret,omitempty"`

	// Username references a secret containing the username (alternative auth)
	// +optional
	Username *corev1.SecretKeySelector `json:"username,omitempty"`

	// Password references a secret containing the password (alternative auth)
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty"`
}

// DNSConfig defines DNS settings for VMs created on this cluster.
type DNSConfig struct {
	// Domain is the default domain for VM FQDNs
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`

	// Servers is a list of DNS servers for VMs
	// If empty, VMs will use DHCP-assigned DNS servers
	// +optional
	Servers []string `json:"servers,omitempty"`

	// RegisterVMs indicates whether to register VMs in external DNS
	// +kubebuilder:default=false
	RegisterVMs bool `json:"registerVMs"`

	// DNSProvider contains configuration for external DNS provider
	// +optional
	DNSProvider *DNSProviderConfig `json:"dnsProvider,omitempty"`
}

// DNSProviderConfig defines external DNS provider configuration.
type DNSProviderConfig struct {
	// Type specifies the DNS provider type (e.g., "route53", "cloudflare")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=route53;cloudflare;azure;gcp
	Type string `json:"type"`

	// Credentials contains provider-specific authentication
	// +kubebuilder:validation:Required
	Credentials DNSProviderCredentials `json:"credentials"`
}

// DNSProviderCredentials defines authentication for DNS providers.
type DNSProviderCredentials struct {
	// AccessKeyID references a secret containing AWS access key ID (Route53)
	// +optional
	AccessKeyID *corev1.SecretKeySelector `json:"accessKeyId,omitempty"`

	// SecretAccessKey references a secret containing AWS secret access key (Route53)
	// +optional
	SecretAccessKey *corev1.SecretKeySelector `json:"secretAccessKey,omitempty"`

	// APIToken references a secret containing API token (Cloudflare, etc.)
	// +optional
	APIToken *corev1.SecretKeySelector `json:"apiToken,omitempty"`
}

// HypervisorClusterStatus defines the observed state of HypervisorCluster.
type HypervisorClusterStatus struct {
	// Conditions represent the latest available observations of the cluster's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ConnectedNodes is the number of nodes currently connected and available
	// +optional
	ConnectedNodes int32 `json:"connectedNodes,omitempty"`

	// AvailableResources represents the total available resources across all nodes
	// +optional
	AvailableResources *ResourceSummary `json:"availableResources,omitempty"`

	// LastSyncTime is the last time the cluster status was synchronized
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// ResourceSummary represents available resources in the hypervisor cluster.
type ResourceSummary struct {
	// CPU represents total available CPU cores
	// +optional
	CPU *resource.Quantity `json:"cpu,omitempty"`

	// Memory represents total available memory
	// +optional
	Memory *resource.Quantity `json:"memory,omitempty"`

	// Storage represents total available storage
	// +optional
	Storage *resource.Quantity `json:"storage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=hyperfleet
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.endpoint"
// +kubebuilder:printcolumn:name="Nodes",type="integer",JSONPath=".status.connectedNodes"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// HypervisorCluster is the Schema for the hypervisorclusters API.
type HypervisorCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HypervisorClusterSpec   `json:"spec,omitempty"`
	Status HypervisorClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HypervisorClusterList contains a list of HypervisorCluster.
type HypervisorClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HypervisorCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HypervisorCluster{}, &HypervisorClusterList{})
}
