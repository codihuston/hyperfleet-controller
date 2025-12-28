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

package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	hypervisorv1alpha1 "github.com/codihuston/hyperfleet-operator/api/v1alpha1"
	"github.com/codihuston/hyperfleet-operator/internal/provider"
)

func TestHypervisorMachineTemplateReconciler_isClusterReady(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *hypervisorv1alpha1.HypervisorCluster
		expected bool
	}{
		{
			name: "cluster is ready",
			cluster: &hypervisorv1alpha1.HypervisorCluster{
				Status: hypervisorv1alpha1.HypervisorClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "cluster is not ready",
			cluster: &hypervisorv1alpha1.HypervisorCluster{
				Status: hypervisorv1alpha1.HypervisorClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: metav1.ConditionFalse,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "cluster has no conditions",
			cluster: &hypervisorv1alpha1.HypervisorCluster{
				Status: hypervisorv1alpha1.HypervisorClusterStatus{
					Conditions: []metav1.Condition{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HypervisorMachineTemplateReconciler{}
			result := r.isClusterReady(tt.cluster)
			if result != tt.expected {
				t.Errorf("isClusterReady() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHypervisorMachineTemplateReconciler_setTemplateValidCondition(t *testing.T) {
	template := &hypervisorv1alpha1.HypervisorMachineTemplate{
		Status: hypervisorv1alpha1.HypervisorMachineTemplateStatus{
			Conditions: []metav1.Condition{},
		},
	}

	r := &HypervisorMachineTemplateReconciler{}

	// Test adding new condition
	r.setTemplateValidCondition(template, metav1.ConditionTrue, "TestReason", "Test message")

	if len(template.Status.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(template.Status.Conditions))
	}

	condition := template.Status.Conditions[0]
	if condition.Type != "TemplateValid" {
		t.Errorf("Expected condition type 'TemplateValid', got %s", condition.Type)
	}
	if condition.Status != metav1.ConditionTrue {
		t.Errorf("Expected condition status True, got %s", condition.Status)
	}
	if condition.Reason != "TestReason" {
		t.Errorf("Expected reason 'TestReason', got %s", condition.Reason)
	}

	// Test updating existing condition
	r.setTemplateValidCondition(template, metav1.ConditionFalse, "UpdatedReason", "Updated message")

	if len(template.Status.Conditions) != 1 {
		t.Errorf("Expected 1 condition after update, got %d", len(template.Status.Conditions))
	}

	condition = template.Status.Conditions[0]
	if condition.Status != metav1.ConditionFalse {
		t.Errorf("Expected updated condition status False, got %s", condition.Status)
	}
	if condition.Reason != "UpdatedReason" {
		t.Errorf("Expected updated reason 'UpdatedReason', got %s", condition.Reason)
	}
}

func TestHypervisorMachineTemplateReconciler_validateWithProvider(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = hypervisorv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name        string
		template    *hypervisorv1alpha1.HypervisorMachineTemplate
		cluster     *hypervisorv1alpha1.HypervisorCluster
		expectError bool
	}{
		{
			name: "valid proxmox template",
			template: &hypervisorv1alpha1.HypervisorMachineTemplate{
				Spec: hypervisorv1alpha1.HypervisorMachineTemplateSpec{
					Template: hypervisorv1alpha1.TemplateSpec{
						Proxmox: &hypervisorv1alpha1.ProxmoxTemplateSpec{
							TemplateID: 9000,
						},
					},
					Resources: hypervisorv1alpha1.ResourceRequirements{
						CPU: 2,
					},
				},
			},
			cluster: &hypervisorv1alpha1.HypervisorCluster{
				Spec: hypervisorv1alpha1.HypervisorClusterSpec{
					Provider: "proxmox",
					Endpoint: "https://test.example.com:8006",
				},
			},
			expectError: false,
		},
		{
			name: "invalid proxmox template ID",
			template: &hypervisorv1alpha1.HypervisorMachineTemplate{
				Spec: hypervisorv1alpha1.HypervisorMachineTemplateSpec{
					Template: hypervisorv1alpha1.TemplateSpec{
						Proxmox: &hypervisorv1alpha1.ProxmoxTemplateSpec{
							TemplateID: 0,
						},
					},
					Resources: hypervisorv1alpha1.ResourceRequirements{
						CPU: 2,
					},
				},
			},
			cluster: &hypervisorv1alpha1.HypervisorCluster{
				Spec: hypervisorv1alpha1.HypervisorClusterSpec{
					Provider: "proxmox",
					Endpoint: "https://test.example.com:8006",
				},
			},
			expectError: true,
		},
		{
			name: "invalid CPU specification",
			template: &hypervisorv1alpha1.HypervisorMachineTemplate{
				Spec: hypervisorv1alpha1.HypervisorMachineTemplateSpec{
					Template: hypervisorv1alpha1.TemplateSpec{
						Proxmox: &hypervisorv1alpha1.ProxmoxTemplateSpec{
							TemplateID: 9000,
						},
					},
					Resources: hypervisorv1alpha1.ResourceRequirements{
						CPU: 0,
					},
				},
			},
			cluster: &hypervisorv1alpha1.HypervisorCluster{
				Spec: hypervisorv1alpha1.HypervisorClusterSpec{
					Provider: "proxmox",
					Endpoint: "https://test.example.com:8006",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &HypervisorMachineTemplateReconciler{
				Client:          client,
				Scheme:          scheme,
				ProviderFactory: provider.NewMockClientFactory(),
			}

			ctx := context.Background()
			err := r.validateWithProvider(ctx, tt.template, tt.cluster)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestHypervisorMachineTemplateReconciler_updateStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = hypervisorv1alpha1.AddToScheme(scheme)

	template := &hypervisorv1alpha1.HypervisorMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Status: hypervisorv1alpha1.HypervisorMachineTemplateStatus{},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(template).WithObjects(template).Build()
	r := &HypervisorMachineTemplateReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	err := r.updateStatus(ctx, template)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if template.Status.LastValidated == nil {
		t.Errorf("Expected LastValidated to be set")
	}
}

func TestHypervisorMachineTemplateReconciler_handleDeletion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = hypervisorv1alpha1.AddToScheme(scheme)

	template := &hypervisorv1alpha1.HypervisorMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-template",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(template).Build()
	r := &HypervisorMachineTemplateReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	result, err := r.handleDeletion(ctx, template)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Errorf("Expected no requeue")
	}

	// Verify finalizer was removed
	if len(template.Finalizers) != 0 {
		t.Errorf("Expected finalizer to be removed, but got %v", template.Finalizers)
	}
}

func TestHypervisorMachineTemplateReconciler_validateTemplate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = hypervisorv1alpha1.AddToScheme(scheme)

	template := &hypervisorv1alpha1.HypervisorMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: hypervisorv1alpha1.HypervisorMachineTemplateSpec{
			HypervisorClusterRef: hypervisorv1alpha1.ObjectReference{
				Name: "test-cluster",
			},
		},
	}

	// Test with missing cluster
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(template).Build()
	r := &HypervisorMachineTemplateReconciler{
		Client:          client,
		Scheme:          scheme,
		ProviderFactory: provider.NewMockClientFactory(),
	}

	ctx := context.Background()
	result, err := r.validateTemplate(ctx, template)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if result.RequeueAfter != TemplateRequeueInterval {
		t.Errorf("Expected requeue after %v, got %v", TemplateRequeueInterval, result.RequeueAfter)
	}

	// Verify condition was set
	found := false
	for _, condition := range template.Status.Conditions {
		if condition.Type == ConditionTemplateValid && condition.Reason == "ClusterNotFound" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ClusterNotFound condition to be set")
	}
}
