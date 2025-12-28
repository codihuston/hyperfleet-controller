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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectReference contains enough information to let you inspect or modify the referred object
type ObjectReference struct {
	// Name of the referent
	Name string `json:"name"`

	// Namespace of the referent, defaults to the same namespace as the referring object
	Namespace string `json:"namespace,omitempty"`
}

// SecretKeySelector selects a key of a Secret
type SecretKeySelector struct {
	// Name of the secret
	Name string `json:"name"`

	// Key within the secret
	Key string `json:"key"`

	// Namespace of the secret, defaults to the same namespace as the referring object
	Namespace string `json:"namespace,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HypervisorMachineTemplateSpec defines the desired state of HypervisorMachineTemplate.
type HypervisorMachineTemplateSpec struct {
	// HypervisorClusterRef references the target hypervisor cluster
	// +kubebuilder:validation:Required
	HypervisorClusterRef ObjectReference `json:"hypervisorClusterRef"`

	// Template defines hypervisor-specific template configuration
	// +kubebuilder:validation:Required
	Template TemplateSpec `json:"template"`

	// Resources defines the VM resource requirements
	// +kubebuilder:validation:Required
	Resources ResourceRequirements `json:"resources"`

	// Attestation configures VM identity verification
	// +kubebuilder:validation:Required
	Attestation AttestationSpec `json:"attestation"`

	// Bootstrap configures workload credential provisioning
	// +kubebuilder:validation:Required
	Bootstrap BootstrapSpec `json:"bootstrap"`

	// Network defines network configuration for VMs
	Network NetworkSpec `json:"network,omitempty"`

	// CloudInit provides custom cloud-init configuration
	CloudInit *CloudInitSpec `json:"cloudInit,omitempty"`
}

// TemplateSpec defines hypervisor-specific template configuration
type TemplateSpec struct {
	// Proxmox-specific template configuration
	Proxmox *ProxmoxTemplateSpec `json:"proxmox,omitempty"`
}

// ProxmoxTemplateSpec defines Proxmox VE template configuration
type ProxmoxTemplateSpec struct {
	// TemplateID is the Proxmox template ID to clone from
	TemplateID int `json:"templateId"`

	// Clone enables VM cloning from template
	Clone bool `json:"clone,omitempty"`

	// LinkedClone enables linked clone for faster provisioning
	LinkedClone bool `json:"linkedClone,omitempty"`
}

// ResourceRequirements defines VM resource specifications
type ResourceRequirements struct {
	// CPU cores for the VM
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=64
	CPU int `json:"cpu"`

	// Memory allocation for the VM (e.g., "4Gi", "8192Mi")
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i?$`
	Memory string `json:"memory"`

	// Disk size for the VM (e.g., "50G", "100G")
	// +kubebuilder:validation:Pattern=`^[0-9]+G$`
	Disk string `json:"disk"`
}

// AttestationSpec configures VM identity verification
type AttestationSpec struct {
	// Method specifies the attestation method (join-token, tpm)
	// +kubebuilder:validation:Enum=join-token;tpm
	Method string `json:"method"`

	// Config provides method-specific configuration
	Config AttestationConfig `json:"config,omitempty"`
}

// AttestationConfig provides attestation method configuration
type AttestationConfig struct {
	// JoinTokenTTL specifies join token time-to-live
	JoinTokenTTL string `json:"joinTokenTTL,omitempty"`

	// TPMDevice specifies TPM device path for TPM attestation
	TPMDevice string `json:"tpmDevice,omitempty"`
}

// BootstrapSpec configures workload credential provisioning
type BootstrapSpec struct {
	// Method specifies the bootstrap method (runner-token, external-secrets)
	// +kubebuilder:validation:Enum=runner-token;external-secrets
	Method string `json:"method"`

	// Config provides method-specific configuration
	Config BootstrapConfig `json:"config,omitempty"`
}

// BootstrapConfig provides bootstrap method configuration
type BootstrapConfig struct {
	// GitHub configuration for runner-token method
	GitHub *GitHubConfig `json:"github,omitempty"`
}

// GitHubConfig defines GitHub-specific bootstrap configuration
type GitHubConfig struct {
	// App configuration for GitHub App credentials (recommended)
	App *GitHubAppConfig `json:"app,omitempty"`

	// PAT configuration for Personal Access Token (development)
	PAT *SecretKeySelector `json:"pat,omitempty"`

	// Repository or organization URL for runner registration
	URL string `json:"url"`

	// Runner configuration
	Runner GitHubRunnerConfig `json:"runner,omitempty"`
}

// GitHubAppConfig defines GitHub App credential configuration
type GitHubAppConfig struct {
	// AppID references the GitHub App ID
	AppID SecretKeySelector `json:"appId"`

	// PrivateKey references the GitHub App private key
	PrivateKey SecretKeySelector `json:"privateKey"`

	// InstallationID references the GitHub App installation ID
	InstallationID SecretKeySelector `json:"installationId"`
}

// GitHubRunnerConfig defines GitHub Actions runner configuration
type GitHubRunnerConfig struct {
	// DownloadURL for GitHub Actions runner binary
	DownloadURL string `json:"downloadUrl,omitempty"`

	// InstallPath for runner installation
	InstallPath string `json:"installPath,omitempty"`

	// WorkDir for runner working directory
	WorkDir string `json:"workDir,omitempty"`

	// Labels for runner registration
	Labels []string `json:"labels,omitempty"`
}

// NetworkSpec defines network configuration
type NetworkSpec struct {
	// Mode specifies network configuration mode (dhcp, static, cloud-init)
	Mode string `json:"mode,omitempty"`

	// StaticConfig provides static IP configuration
	StaticConfig *StaticNetworkConfig `json:"staticConfig,omitempty"`

	// DHCP provides DHCP-specific configuration
	DHCP *DHCPConfig `json:"dhcp,omitempty"`
}

// StaticNetworkConfig defines static IP configuration
type StaticNetworkConfig struct {
	// IP address with CIDR notation
	IP string `json:"ip"`

	// Gateway IP address
	Gateway string `json:"gateway"`

	// DNS servers
	DNS []string `json:"dns,omitempty"`
}

// DHCPConfig defines DHCP-specific configuration
type DHCPConfig struct {
	// SendHostname enables sending hostname in DHCP request
	SendHostname bool `json:"sendHostname,omitempty"`

	// RequestStaticLease requests same IP for hostname
	RequestStaticLease bool `json:"requestStaticLease,omitempty"`

	// ClientIdentifier specifies DHCP client identifier
	ClientIdentifier string `json:"clientIdentifier,omitempty"`
}

// CloudInitSpec defines custom cloud-init configuration
type CloudInitSpec struct {
	// UserData provides cloud-init user data
	UserData string `json:"userData,omitempty"`

	// MetaData provides cloud-init meta data
	MetaData string `json:"metaData,omitempty"`
}

// HypervisorMachineTemplateStatus defines the observed state of HypervisorMachineTemplate.
type HypervisorMachineTemplateStatus struct {
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// TemplateAvailable indicates if the referenced template exists
	TemplateAvailable bool `json:"templateAvailable,omitempty"`

	// ValidationStatus indicates template validation result
	ValidationStatus string `json:"validationStatus,omitempty"`

	// LastValidated timestamp of last validation
	LastValidated *metav1.Time `json:"lastValidated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Template ID",type=integer,JSONPath=`.spec.template.proxmox.templateId`
// +kubebuilder:printcolumn:name="CPU",type=integer,JSONPath=`.spec.resources.cpu`
// +kubebuilder:printcolumn:name="Memory",type=string,JSONPath=`.spec.resources.memory`
// +kubebuilder:printcolumn:name="Available",type=boolean,JSONPath=`.status.templateAvailable`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HypervisorMachineTemplate is the Schema for the hypervisormachinetemplates API.
type HypervisorMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HypervisorMachineTemplateSpec   `json:"spec,omitempty"`
	Status HypervisorMachineTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HypervisorMachineTemplateList contains a list of HypervisorMachineTemplate.
type HypervisorMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HypervisorMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HypervisorMachineTemplate{}, &HypervisorMachineTemplateList{})
}
