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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hypervisorv1alpha1 "github.com/codihuston/hyperfleet-operator/api/v1alpha1"
	"github.com/codihuston/hyperfleet-operator/internal/provider"
)

var _ = Describe("HypervisorMachineTemplate Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		hypervisormachinetemplate := &hypervisorv1alpha1.HypervisorMachineTemplate{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind HypervisorMachineTemplate")
			err := k8sClient.Get(ctx, typeNamespacedName, hypervisormachinetemplate)
			if err != nil && errors.IsNotFound(err) {
				resource := &hypervisorv1alpha1.HypervisorMachineTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: hypervisorv1alpha1.HypervisorMachineTemplateSpec{
						HypervisorClusterRef: hypervisorv1alpha1.ObjectReference{
							Name: "test-cluster",
						},
						Template: hypervisorv1alpha1.TemplateSpec{
							Proxmox: &hypervisorv1alpha1.ProxmoxTemplateSpec{
								TemplateID:  9000,
								Clone:       true,
								LinkedClone: true,
							},
						},
						Resources: hypervisorv1alpha1.ResourceRequirements{
							CPU:    2,
							Memory: "4Gi",
							Disk:   "20G",
						},
						Attestation: hypervisorv1alpha1.AttestationSpec{
							Method: "join-token",
							Config: hypervisorv1alpha1.AttestationConfig{
								JoinTokenTTL: "1h",
							},
						},
						Bootstrap: hypervisorv1alpha1.BootstrapSpec{
							Method: "runner-token",
							Config: hypervisorv1alpha1.BootstrapConfig{
								GitHub: &hypervisorv1alpha1.GitHubConfig{
									URL: "https://github.com/test/repo",
									PAT: &hypervisorv1alpha1.SecretKeySelector{
										Name: "github-pat",
										Key:  "token",
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &hypervisorv1alpha1.HypervisorMachineTemplate{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance HypervisorMachineTemplate")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HypervisorMachineTemplateReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				ProviderFactory: provider.NewMockClientFactory(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle deletion with finalizers", func() {
			By("Creating a template with finalizer")
			template := &hypervisorv1alpha1.HypervisorMachineTemplate{}
			err := k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &HypervisorMachineTemplateReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				ProviderFactory: provider.NewMockClientFactory(),
			}

			By("Adding finalizer through reconciliation")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying finalizer was added")
			err = k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(err).NotTo(HaveOccurred())
			Expect(template.Finalizers).To(ContainElement("hypervisormachinetemplate.hyperfleet.io/finalizer"))

			By("Deleting the template")
			err = k8sClient.Delete(ctx, template)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling deletion")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying template is deleted")
			err = k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should validate template against cluster", func() {
			By("Creating a HypervisorCluster")
			cluster := &hypervisorv1alpha1.HypervisorCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: hypervisorv1alpha1.HypervisorClusterSpec{
					Provider: "proxmox",
					Endpoint: "https://test.example.com:8006",
					Credentials: hypervisorv1alpha1.HypervisorCredentials{
						TokenID: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "token-id",
						},
						TokenSecret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "token-secret",
						},
					},
					Nodes:          []string{"node1"},
					DefaultStorage: "local-lvm",
					DefaultNetwork: "vmbr0",
				},
				Status: hypervisorv1alpha1.HypervisorClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: metav1.ConditionTrue,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Reconciling the template")
			controllerReconciler := &HypervisorMachineTemplateReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				ProviderFactory: provider.NewMockClientFactory(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying template status is updated")
			template := &hypervisorv1alpha1.HypervisorMachineTemplate{}
			err = k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(err).NotTo(HaveOccurred())
			Expect(template.Status.Conditions).NotTo(BeEmpty())
		})

		It("should handle missing cluster reference", func() {
			By("Creating template with non-existent cluster reference")
			template := &hypervisorv1alpha1.HypervisorMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-missing-cluster",
					Namespace: "default",
				},
				Spec: hypervisorv1alpha1.HypervisorMachineTemplateSpec{
					HypervisorClusterRef: hypervisorv1alpha1.ObjectReference{
						Name: "non-existent-cluster",
					},
					Template: hypervisorv1alpha1.TemplateSpec{
						Proxmox: &hypervisorv1alpha1.ProxmoxTemplateSpec{
							TemplateID:  9000,
							Clone:       true,
							LinkedClone: true,
						},
					},
					Resources: hypervisorv1alpha1.ResourceRequirements{
						CPU:    2,
						Memory: "4Gi",
						Disk:   "20G",
					},
					Attestation: hypervisorv1alpha1.AttestationSpec{
						Method: "join-token",
					},
					Bootstrap: hypervisorv1alpha1.BootstrapSpec{
						Method: "runner-token",
						Config: hypervisorv1alpha1.BootstrapConfig{
							GitHub: &hypervisorv1alpha1.GitHubConfig{
								URL: "https://github.com/test/repo",
								PAT: &hypervisorv1alpha1.SecretKeySelector{
									Name: "github-pat",
									Key:  "token",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, template)).To(Succeed())

			By("Reconciling the template")
			controllerReconciler := &HypervisorMachineTemplateReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				ProviderFactory: provider.NewMockClientFactory(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-missing-cluster",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying template has ClusterNotFound condition")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-missing-cluster",
				Namespace: "default",
			}, template)
			Expect(err).NotTo(HaveOccurred())

			found := false
			for _, condition := range template.Status.Conditions {
				if condition.Type == "TemplateValid" && condition.Reason == "ClusterNotFound" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})
})
