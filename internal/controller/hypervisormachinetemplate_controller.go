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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	hypervisorv1alpha1 "github.com/codihuston/hyperfleet-operator/api/v1alpha1"
	"github.com/codihuston/hyperfleet-operator/internal/provider"
)

// HypervisorMachineTemplateReconciler reconciles a HypervisorMachineTemplate object
type HypervisorMachineTemplateReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ProviderFactory provider.ClientFactory
}

const (
	// FinalizerName is the finalizer used by this controller
	FinalizerName = "hypervisormachinetemplate.hyperfleet.io/finalizer"

	// TemplateRequeueInterval for periodic validation checks
	TemplateRequeueInterval = 5 * time.Minute

	// DefaultProviderTimeout for hypervisor client operations
	DefaultProviderTimeout = 300 // 5 minutes in seconds

	// ConditionTemplateValid represents the template validation condition
	ConditionTemplateValid = "TemplateValid"
)

// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisormachinetemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisormachinetemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisormachinetemplates/finalizers,verbs=update
// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisorclusters,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HypervisorMachineTemplate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *HypervisorMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the HypervisorMachineTemplate instance
	template := &hypervisorv1alpha1.HypervisorMachineTemplate{}
	if err := r.Get(ctx, req.NamespacedName, template); err != nil {
		if errors.IsNotFound(err) {
			log.Info("HypervisorMachineTemplate resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get HypervisorMachineTemplate")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !template.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, template)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(template, FinalizerName) {
		controllerutil.AddFinalizer(template, FinalizerName)
		if err := r.Update(ctx, template); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate template against hypervisor
	result, err := r.validateTemplate(ctx, template)
	if err != nil {
		log.Error(err, "Failed to validate template")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, template); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypervisorMachineTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hypervisorv1alpha1.HypervisorMachineTemplate{}).
		Named("hypervisormachinetemplate").
		Complete(r)
}

// handleDeletion handles the deletion of HypervisorMachineTemplate resources
func (r *HypervisorMachineTemplateReconciler) handleDeletion(ctx context.Context, template *hypervisorv1alpha1.HypervisorMachineTemplate) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Perform cleanup logic here if needed
	log.Info("Cleaning up HypervisorMachineTemplate", "name", template.Name)

	// Remove finalizer
	controllerutil.RemoveFinalizer(template, FinalizerName)
	if err := r.Update(ctx, template); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// validateTemplate validates the template against the hypervisor
func (r *HypervisorMachineTemplateReconciler) validateTemplate(ctx context.Context, template *hypervisorv1alpha1.HypervisorMachineTemplate) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Get the referenced HypervisorCluster
	cluster := &hypervisorv1alpha1.HypervisorCluster{}
	clusterKey := client.ObjectKey{
		Name:      template.Spec.HypervisorClusterRef.Name,
		Namespace: template.Spec.HypervisorClusterRef.Namespace,
	}
	if clusterKey.Namespace == "" {
		clusterKey.Namespace = template.Namespace
	}

	if err := r.Get(ctx, clusterKey, cluster); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Referenced HypervisorCluster not found", "cluster", clusterKey)
			r.setTemplateValidCondition(template, metav1.ConditionFalse, "ClusterNotFound", "Referenced HypervisorCluster not found")
			return ctrl.Result{RequeueAfter: TemplateRequeueInterval}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if cluster is ready
	if !r.isClusterReady(cluster) {
		log.Info("Referenced HypervisorCluster not ready", "cluster", clusterKey)
		r.setTemplateValidCondition(template, metav1.ConditionFalse, "ClusterNotReady", "Referenced HypervisorCluster is not ready")
		return ctrl.Result{RequeueAfter: TemplateRequeueInterval}, nil
	}

	// Create provider client and validate template
	if err := r.validateWithProvider(ctx, template, cluster); err != nil {
		log.Error(err, "Template validation failed")
		r.setTemplateValidCondition(template, metav1.ConditionFalse, "ValidationFailed", err.Error())
		return ctrl.Result{RequeueAfter: TemplateRequeueInterval}, nil
	}

	// Template is valid
	r.setTemplateValidCondition(template, metav1.ConditionTrue, "ValidationSucceeded", "Template validation succeeded")
	template.Status.TemplateAvailable = true
	template.Status.ValidationStatus = "Valid"

	return ctrl.Result{RequeueAfter: TemplateRequeueInterval}, nil
}

// validateWithProvider validates the template using the hypervisor provider
func (r *HypervisorMachineTemplateReconciler) validateWithProvider(_ context.Context, template *hypervisorv1alpha1.HypervisorMachineTemplate, cluster *hypervisorv1alpha1.HypervisorCluster) error {
	// Create provider client configuration
	clientConfig := &provider.ClientConfig{
		Endpoint: cluster.Spec.Endpoint,
		Timeout:  DefaultProviderTimeout,
	}

	// Create auth config (simplified for now - would need to read from secrets in real implementation)
	authConfig := &provider.AuthConfig{
		Type: "token", // Default to token auth for Proxmox
	}

	// Create provider client
	providerClient, err := r.ProviderFactory.CreateClient(cluster.Spec.Provider, clientConfig, authConfig)
	if err != nil {
		return fmt.Errorf("failed to create provider client: %w", err)
	}
	defer func() {
		_ = providerClient.Close() // Ignore close errors in validation
	}()

	// For Proxmox, validate that the template configuration is valid
	if template.Spec.Template.Proxmox != nil {
		if template.Spec.Template.Proxmox.TemplateID <= 0 {
			return fmt.Errorf("invalid Proxmox template ID: %d", template.Spec.Template.Proxmox.TemplateID)
		}

		// TODO: Add actual template existence check via provider client
		// This would call something like: providerClient.ValidateTemplate(templateID)
	}

	// Validate resource requirements
	if template.Spec.Resources.CPU <= 0 {
		return fmt.Errorf("invalid CPU specification: %d", template.Spec.Resources.CPU)
	}

	return nil
}

// isClusterReady checks if the HypervisorCluster is ready
func (r *HypervisorMachineTemplateReconciler) isClusterReady(cluster *hypervisorv1alpha1.HypervisorCluster) bool {
	for _, condition := range cluster.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// setTemplateValidCondition sets the TemplateValid condition on the template status
func (r *HypervisorMachineTemplateReconciler) setTemplateValidCondition(template *hypervisorv1alpha1.HypervisorMachineTemplate, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               ConditionTemplateValid,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Find existing condition and update or append
	for i, existingCondition := range template.Status.Conditions {
		if existingCondition.Type == ConditionTemplateValid {
			template.Status.Conditions[i] = condition
			return
		}
	}
	template.Status.Conditions = append(template.Status.Conditions, condition)
}

// updateStatus updates the template status
func (r *HypervisorMachineTemplateReconciler) updateStatus(ctx context.Context, template *hypervisorv1alpha1.HypervisorMachineTemplate) error {
	now := metav1.Now()
	template.Status.LastValidated = &now

	return r.Status().Update(ctx, template)
}
