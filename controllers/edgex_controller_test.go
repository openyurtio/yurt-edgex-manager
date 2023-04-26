/*

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
// +kubebuilder:docs-gen:collapse=Apache License

/*
Ideally, we should have one `<kind>_controller_test.go` for each controller scaffolded and called in the `suite_test.go`.
So, let's write our example test for the CronJob controller (`cronjob_controller_test.go.`)
*/

/*
As usual, we start with the necessary imports. We also define some utility variables.
*/
package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	unitv1alpha1 "github.com/openyurtio/api/apps/v1alpha1"
	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("EdgeX controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		EdgexName      = "test-edgex"
		EdgexNamespace = "default"
		EdgexVersion   = "jakarta"

		PoolName      = "pool-bj"
		PoolNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating EdgeX Status", func() {
		It("Should trigger EdgeX instance", func() {
			By("By creating a new EdgeX deployment")
			ctx := context.Background()

			pool := &unitv1alpha1.NodePool{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps.openyurt.io/v1alpha1",
					Kind:       "NodePool",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      PoolName,
					Namespace: PoolNamespace,
				},
				Spec: unitv1alpha1.NodePoolSpec{
					Type: "Cloud",
				},
			}
			Expect(k8sClient.Create(ctx, pool)).Should(Succeed())

			edgex := &v1alpha1.EdgeX{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "device.openyurt.io/v1alpha1",
					Kind:       "EdgeX",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      EdgexName,
					Namespace: EdgexNamespace,
				},
				Spec: v1alpha1.EdgeXSpec{
					Version:  EdgexVersion,
					PoolName: PoolName,
				},
			}
			Expect(k8sClient.Create(ctx, edgex)).Should(Succeed())

			edgexLookupKey := types.NamespacedName{Name: EdgexName, Namespace: EdgexNamespace}
			createdEdgex := &v1alpha1.EdgeX{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, edgexLookupKey, createdEdgex)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			time.Sleep(10 * time.Second)
			Expect(createdEdgex.Spec.Version).Should(Equal(EdgexVersion))

			Expect(k8sClient.Delete(ctx, createdEdgex)).Should(Succeed())
		})
	})

})
