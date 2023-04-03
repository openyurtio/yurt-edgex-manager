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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("test webhook", func() {
	var (
		ctx       = context.TODO()
		specName  = "edgex webhook"
		edgexes   *devicev1alpha1.EdgeXList
		k8sClient client.Client
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(ClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		edgexes = &devicev1alpha1.EdgeXList{}
		k8sClient = ClusterProxy.GetClient()
	})

	AfterEach(func() {
		By("after a webhook test, clean up previous resources")
		cleanupEdgex(ctx, k8sClient, edgexes)
	})

	It("Create a edgex without setting version and servicetype", func() {
		edgexForDefault := &devicev1alpha1.EdgeX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "edgex-webhook-beijing",
				Namespace: "default",
			},
			Spec: devicev1alpha1.EdgeXSpec{
				PoolName: "beijing",
			},
		}
		k8sClient.Create(ctx, edgexForDefault)
		resForDefault := &devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-beijing",
			}
			if err := k8sClient.Get(ctx, key, resForDefault); err != nil {
				return false
			}
			if resForDefault.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgexForDefault)
				By("edgex create in beijing")
				return true
			}
			return false

		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeTrue(), func() string { return "EdgeX beijing without setting version and servicetype not ready" })

		By("Create a edgex with an already occupied nodepool")
		edgexForOccupiedNodePool := &devicev1alpha1.EdgeX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "edgex-webhook-occupied-beijing",
				Namespace: "default",
			},
			Spec: devicev1alpha1.EdgeXSpec{
				PoolName: "beijing",
			},
		}
		k8sClient.Create(ctx, edgexForDefault)
		resForOccupiedNodePool := &devicev1alpha1.EdgeX{}

		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-occupied-beijing",
			}
			if err := k8sClient.Get(ctx, key, resForOccupiedNodePool); err != nil {
				return false
			}
			if resForOccupiedNodePool.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgexForOccupiedNodePool)
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX beijing with an already occupied nodepool ready" })
	})

	It("Create a edgex in beijing with wrong servicetype", func() {
		edgexForWrongServiceType := &devicev1alpha1.EdgeX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "edgex-webhook-beijing-servicetype",
				Namespace: "default",
			},
			Spec: devicev1alpha1.EdgeXSpec{
				PoolName:    "beijing",
				ServiceType: "test",
			},
		}
		k8sClient.Create(ctx, edgexForWrongServiceType)
		res := &devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-beijing-servicetype",
			}
			if err := k8sClient.Get(ctx, key, res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgexForWrongServiceType)
				return true
			}
			return false

		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX beijing with wrong servicetype ready" })
	})

	It("Create a edgex in beijing with wrong poolname", func() {
		edgexForWrongPoolName := &devicev1alpha1.EdgeX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "edgex-webhook-shanghai-poolname",
				Namespace: "default",
			},
			Spec: devicev1alpha1.EdgeXSpec{
				PoolName: "shanghai",
			},
		}
		k8sClient.Create(ctx, edgexForWrongPoolName)
		res := &devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "edgex-webhook-shanghai-poolname",
			}
			if err := k8sClient.Get(ctx, key, res); err != nil {
				return false
			}
			if res.Status.Ready == true {
				edgexes.Items = append(edgexes.Items, *edgexForWrongPoolName)
				return true
			}
			return false

		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeFalse(), func() string { return "EdgeX shanghai with wrong poolname ready" })
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
