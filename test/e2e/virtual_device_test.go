/*
Copyright 2022 Wuming Liu.

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
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
)

var _ = Describe("When create virtual device [PR-Blocking]", func() {

	var (
		ctx      context.Context
		specName = "virtual device"
		path     string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(ClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		path = ""
	})

	AfterEach(func() {
		By("after a virtual device test, clean up previous resources")
		if path != "" {
			ClusterProxy.Delete(ctx, path)
			//time.Sleep(time.Minute)
		}

	})

	It("Create a hanoi edgex in hangzhou", func() {
		ClusterProxy.Apply(ctx, "./data/hangzhou.yaml")
		path = "./data/hangzhou.yaml"
		edgex := devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "hangzhou",
				Name:      "edgex-sample-hangzhou",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &edgex); err != nil {
				return false
			}
			if edgex.Status.Ready == true {
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeTrue(), func() string { return "Edgex hangzhou not ready" })

	})

	It("Create a jakarta edgex in beijing", func() {
		ClusterProxy.Apply(ctx, "./data/beijing.yaml")
		path = "./data/beijing.yaml"
		edgex := devicev1alpha1.EdgeX{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "beijing",
				Name:      "edgex-sample-beijing",
			}
			if err := ClusterProxy.GetClient().Get(ctx, key, &edgex); err != nil {
				return false
			}
			if edgex.Status.Ready == true {
				return true
			}
			return false
		}, e2eConfig.GetIntervals("default", "create-edgex")...).Should(BeTrue(), func() string { return "Edgex beijing not ready" })
	})
})
