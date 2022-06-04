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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
)

var _ = Describe("When create virtual device [PR-Blocking]", func() {

	var (
		ctx      context.Context
		specName = "virtual device"
	)

	BeforeEach(func() {
		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(ClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
	})

	It("Create a virtual device", func() {
		edgex := devicev1alpha1.EdgeX{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "default",
				Name:      "edgex-sample-hangzhou",
			},
			Spec: devicev1alpha1.EdgeXSpec{
				Version:  "hanoi",
				PoolName: "hangzhou",
			},
		}
		Expect(ClusterProxy.GetClient().Create(ctx, &edgex)).To(BeNil(), "Failt to create EdgeX")

		Eventually(func() bool {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      edgex.Name,
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
})
