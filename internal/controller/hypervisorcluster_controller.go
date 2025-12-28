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
	"crypto/tls"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hypervisorv1alpha1 "github.com/codihuston/hyperfleet-operator/api/v1alpha1"
	"github.com/codihuston/hyperfleet-operator/internal/provider"
)

const (
	// RequeueInterval defines how often to requeue reconciliation for periodic connection checks
	RequeueInterval = 5 * time.Minute
	// DefaultTimeout defines the default timeout for hypervisor client operations
	DefaultTimeout = 300 // 5 minutes in seconds
	// DefaultInsecureSkipVerify defines the default TLS verification behavior
	// Set to false by default for security - users must explicitly configure insecure connections
	DefaultInsecureSkipVerify = false

	// ConditionReady represents the ready condition type
	ConditionReady = "Ready"
)

// HypervisorClusterReconciler reconciles a HypervisorCluster object
type HypervisorClusterReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ClientFactory provider.ClientFactory
}

// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisorclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisorclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hypervisor.hyperfleet.io,resources=hypervisorclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *HypervisorClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the HypervisorCluster instance
	var hypervisorCluster hypervisorv1alpha1.HypervisorCluster
	if err := r.Get(ctx, req.NamespacedName, &hypervisorCluster); err != nil {
		if errors.IsNotFound(err) {
			// Resource was deleted
			logger.Info("HypervisorCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get HypervisorCluster")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling HypervisorCluster", "name", hypervisorCluster.Name)

	// Test connection to hypervisor
	connectionResult := r.testConnection(ctx, &hypervisorCluster)

	// Update status based on connection result
	if err := r.updateStatus(ctx, &hypervisorCluster, connectionResult); err != nil {
		logger.Error(err, "Failed to update HypervisorCluster status")
		return ctrl.Result{}, err
	}

	// Requeue after defined interval to periodically check connection
	return ctrl.Result{RequeueAfter: RequeueInterval}, nil
}

// testConnection tests the connection to the hypervisor using the provider adapter
func (r *HypervisorClusterReconciler) testConnection(ctx context.Context, cluster *hypervisorv1alpha1.HypervisorCluster) *ConnectionResult {
	logger := log.FromContext(ctx)

	result := &ConnectionResult{
		Success:  false,
		Message:  "",
		TestedAt: metav1.Now(),
	}

	// Load credentials from secrets
	auth, err := r.loadCredentials(ctx, cluster)
	if err != nil {
		result.Message = fmt.Sprintf("Credential loading failed: %v", err)
		logger.Error(err, "Credential loading failed")
		return result
	}

	// Create client configuration with secure TLS defaults
	// #nosec G402 -- User-configurable TLS with secure defaults (defaults to false)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: DefaultInsecureSkipVerify, // Secure by default
	}

	// Apply user-specified TLS configuration if provided
	if cluster.Spec.TLS != nil {
		tlsConfig.InsecureSkipVerify = cluster.Spec.TLS.InsecureSkipVerify

		// TODO: Implement CA certificate loading from cluster.Spec.TLS.CACertificate
		// This will be added in a future iteration to support custom CA certificates
	}

	clientConfig := &provider.ClientConfig{
		Endpoint:  cluster.Spec.Endpoint,
		TLSConfig: tlsConfig,
		Timeout:   DefaultTimeout,
	}

	// Create hypervisor client using the factory
	if r.ClientFactory == nil {
		r.ClientFactory = provider.NewClientFactory()
	}

	hypervisorClient, err := r.ClientFactory.CreateClient(cluster.Spec.Provider, clientConfig, auth)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create hypervisor client: %v", err)
		logger.Error(err, "Failed to create hypervisor client", "provider", cluster.Spec.Provider)
		return result
	}
	defer func() {
		if closeErr := hypervisorClient.Close(); closeErr != nil {
			logger.Error(closeErr, "Failed to close hypervisor client")
		}
	}()

	// Test the connection
	connInfo, err := hypervisorClient.TestConnection(ctx)
	if err != nil {
		result.Message = fmt.Sprintf("Hypervisor connection failed: %v", err)
		logger.Error(err, "Hypervisor connection failed", "endpoint", cluster.Spec.Endpoint)
		return result
	}

	// If we get here, connection is working
	result.Success = true
	result.Message = fmt.Sprintf("Successfully connected to %s cluster", cluster.Spec.Provider)
	logger.Info("Hypervisor connection test successful",
		"provider", cluster.Spec.Provider,
		"version", connInfo.Version,
		"endpoint", cluster.Spec.Endpoint)

	return result
}

// loadCredentials loads authentication credentials from Kubernetes secrets
func (r *HypervisorClusterReconciler) loadCredentials(ctx context.Context, cluster *hypervisorv1alpha1.HypervisorCluster) (*provider.AuthConfig, error) {
	creds := cluster.Spec.Credentials

	// Check token-based authentication (preferred)
	if creds.TokenID != nil && creds.TokenSecret != nil {
		tokenID, err := r.getSecretValue(ctx, cluster.Namespace, creds.TokenID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tokenId: %w", err)
		}

		tokenSecret, err := r.getSecretValue(ctx, cluster.Namespace, creds.TokenSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get tokenSecret: %w", err)
		}

		return &provider.AuthConfig{
			Type:        "token",
			TokenID:     tokenID,
			TokenSecret: tokenSecret,
		}, nil
	}

	// Check username/password authentication
	if creds.Username != nil && creds.Password != nil {
		username, err := r.getSecretValue(ctx, cluster.Namespace, creds.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to get username: %w", err)
		}

		password, err := r.getSecretValue(ctx, cluster.Namespace, creds.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to get password: %w", err)
		}

		return &provider.AuthConfig{
			Type:     "password",
			Username: username,
			Password: password,
		}, nil
	}

	return nil, fmt.Errorf("no valid credential configuration found")
}

// getSecretValue retrieves a value from a Kubernetes secret
func (r *HypervisorClusterReconciler) getSecretValue(ctx context.Context, namespace string, selector *corev1.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      selector.Name,
		Namespace: namespace,
	}

	if err := r.Get(ctx, secretName, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	value, exists := secret.Data[selector.Key]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret %s", selector.Key, secretName)
	}

	return string(value), nil
}

// updateStatus updates the HypervisorCluster status based on connection test results
func (r *HypervisorClusterReconciler) updateStatus(ctx context.Context, cluster *hypervisorv1alpha1.HypervisorCluster, result *ConnectionResult) error {
	// Update last sync time
	cluster.Status.LastSyncTime = &result.TestedAt

	// Prepare condition
	condition := metav1.Condition{
		Type:               ConditionReady,
		LastTransitionTime: result.TestedAt,
		ObservedGeneration: cluster.Generation,
		Message:            result.Message,
	}

	if result.Success {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "ConnectionSuccessful"
	} else {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "ConnectionFailed"
	}

	// Update or add the condition
	updated := false
	for i, existingCondition := range cluster.Status.Conditions {
		if existingCondition.Type == ConditionReady {
			cluster.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		cluster.Status.Conditions = append(cluster.Status.Conditions, condition)
	}

	// Update the status
	return r.Status().Update(ctx, cluster)
}

// ConnectionResult holds the result of a connection test
type ConnectionResult struct {
	Success  bool
	Message  string
	TestedAt metav1.Time
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypervisorClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hypervisorv1alpha1.HypervisorCluster{}).
		Named("hypervisorcluster").
		Complete(r)
}
