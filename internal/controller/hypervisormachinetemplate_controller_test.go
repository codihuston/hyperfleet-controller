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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hypervisorv1alpha1 "github.com/codihuston/hyperfleet-operator/api/v1alpha1"
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
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
