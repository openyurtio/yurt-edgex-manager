/*
Copyright 2022.

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

package e2e

import (
	"context"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("test webhook", func() {
	var (
		ctx      context.Context
		specName = "edgex webhook"

		edgexes   = &devicev1alpha1.EdgeXList{}
		edgex     *devicev1alpha1.EdgeX
		k8sClient client.Client
		mutex     sync.Mutex
	)

	BeforeEach(func() {
		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(ClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		edgex = &devicev1alpha1.EdgeX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "edgex-webhook-beijing",
				Namespace: "default",
			},
			Spec: devicev1alpha1.EdgeXSpec{
				PoolName: "beijing",
			},
		}
		k8sClient = ClusterProxy.GetClient()
	})

	AfterEach(func() {
		By("after a webhook test, clean up previous resources")
		cleanupEdgex(ctx, ClusterProxy.GetClient(), edgexes)
	})

	It("Create a edgex in beijing with wrong version", func() {
		edgex.ObjectMeta.Name += "-version"
		edgex.Spec.Version = "test"
		k8sClient.Create(ctx, edgex)
		res := devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-beijing-version",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgex)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX beijing with swrong version ready" })
	})

	It("Create a edgex in beijing with wrong servicetype", func() {
		edgex.ObjectMeta.Name += "-servicetype"
		edgex.Spec.ServiceType = "test"
		k8sClient.Create(ctx, edgex)
		res := devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-beijing-servicetype",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgex)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX beijing with wrong servicetype ready" })
	})

	It("Create a edgex in beijing with wrong poolname", func() {
		edgex.ObjectMeta.Name += "-poolname"
		edgex.Spec.PoolName = "shanghai"
		k8sClient.Create(ctx, edgex)
		res := devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-beijing-poolname",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgex)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX beijing with wrong poolname ready" })
	})

	It("Create a edgex without setting version and servicetype", func() {
		edgex2 := edgex.DeepCopy()
		edgex3 := edgex.DeepCopy()
		k8sClient.Create(ctx, edgex)
		res := devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-beijing",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				mutex.Lock()
				defer mutex.Unlock()
				edgexes.Items = append(edgexes.Items, *edgex)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeTrue(), func() string { return "EdgeX beijing without setting version and servicetype not ready" })

		By("Create a edgex with an already occupied nodepool")
		edgex2.ObjectMeta.Name = "edgex2-webhook-beijing"
		k8sClient.Create(ctx, edgex2)
		res = devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex2-webhook-beijing",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				mutex.Lock()
				defer mutex.Unlock()
				edgexes.Items = append(edgexes.Items, *edgex2)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX beijing with an already occupied nodepool ready" })

		By("Create a edgex in hangzhou")
		edgex3.ObjectMeta.Name = "edgex-webhook-hangzhou"
		edgex3.Spec.PoolName = "hangzhou"
		k8sClient.Create(ctx, edgex3)
		res = devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-hangzhou",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				mutex.Lock()
				defer mutex.Unlock()
				edgexes.Items = append(edgexes.Items, *edgex3)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeTrue(), func() string { return "EdgeX hangzhou not ready" })

	})
})

func cleanupEdgex(ctx context.Context, k8sClient client.Client, edgexes *devicev1alpha1.EdgeXList) error {
	for _, item := range edgexes.Items {
		if err := k8sClient.Delete(ctx, &item); err != nil {
			return err
		}
	}
	return nil
}
