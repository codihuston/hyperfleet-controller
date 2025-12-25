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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hypervisorv1alpha1 "github.com/codihuston/hyperfleet-operator/api/v1alpha1"
)

var _ = Describe("HypervisorCluster Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const secretName = "test-credentials"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		secretNamespacedName := types.NamespacedName{
			Name:      secretName,
			Namespace: "default",
		}

		BeforeEach(func() {
			By("creating the credentials secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"tokenId":     []byte("test-token-id"),
					"tokenSecret": []byte("test-token-secret"),
				},
			}
			err := k8sClient.Create(ctx, secret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the custom resource for the Kind HypervisorCluster")
			hypervisorcluster := &hypervisorv1alpha1.HypervisorCluster{}
			err = k8sClient.Get(ctx, typeNamespacedName, hypervisorcluster)
			if err != nil && apierrors.IsNotFound(err) {
				resource := &hypervisorv1alpha1.HypervisorCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: hypervisorv1alpha1.HypervisorClusterSpec{
						Provider: "proxmox",
						Endpoint: "https://pve.example.com:8006/api2/json",
						Credentials: hypervisorv1alpha1.HypervisorCredentials{
							TokenID: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secretName,
								},
								Key: "tokenId",
							},
							TokenSecret: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secretName,
								},
								Key: "tokenSecret",
							},
						},
						Nodes:          []string{"pve-node-1", "pve-node-2"},
						DefaultStorage: "local-lvm",
						DefaultNetwork: "vmbr0",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the HypervisorCluster resource")
			resource := &hypervisorv1alpha1.HypervisorCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			By("Cleanup the credentials secret")
			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, secretNamespacedName, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorClusterReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the resource status is updated")
			resource := &hypervisorv1alpha1.HypervisorCluster{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			// Check that status was updated (will fail connection but should update status)
			Expect(resource.Status.LastSyncTime).NotTo(BeNil())

			// Should have at least one condition
			Expect(len(resource.Status.Conditions)).To(BeNumerically(">", 0))

			// Find the Ready condition
			var readyCondition *metav1.Condition
			for i := range resource.Status.Conditions {
				if resource.Status.Conditions[i].Type == "Ready" {
					readyCondition = &resource.Status.Conditions[i]
					break
				}
			}
			Expect(readyCondition).NotTo(BeNil())
			// Connection will likely fail in test environment, so we just verify the condition exists
			Expect(readyCondition.Status).To(Or(Equal(metav1.ConditionTrue), Equal(metav1.ConditionFalse)))
		})
	})
})
