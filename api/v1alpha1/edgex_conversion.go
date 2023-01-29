/*
Copyright 2021.
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

package v1alpha1

import (
	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *EdgeX) ConvertTo(dstRaw conversion.Hub) error {
	// Transform metadata
	dst := dstRaw.(*v1alpha2.EdgeX)
	dst.ObjectMeta = src.ObjectMeta
	dst.TypeMeta = src.TypeMeta
	dst.TypeMeta.APIVersion = "device.openyurt.io/v1alpha2"

	// Transform spec
	dst.Spec.Version = src.Spec.Version
	dst.Spec.Security = false
	dst.Spec.ImageRegistry = src.Spec.ImageRegistry
	dst.Spec.PoolName = src.Spec.PoolName

	// Transform status
	dst.Status.Ready = src.Status.Ready
	dst.Status.Initialized = src.Status.Initialized
	dst.Status.ReadyComponentNum = src.Status.DeploymentReadyReplicas
	dst.Status.UnreadyComponentNum = src.Status.DeploymentReplicas - src.Status.DeploymentReadyReplicas
	dst.Status.Conditions = src.Status.Conditions

	//TODO: Components
	return nil
}
func (dst *EdgeX) ConvertFrom(srcRaw conversion.Hub) error {
	// Transform metadata
	src := srcRaw.(*v1alpha2.EdgeX)
	dst.ObjectMeta = src.ObjectMeta
	dst.TypeMeta = src.TypeMeta
	dst.TypeMeta.APIVersion = "device.openyurt.io/v1alpha1"

	// Transform spec
	dst.Spec.Version = src.Spec.Version
	dst.Spec.ImageRegistry = src.Spec.ImageRegistry
	dst.Spec.PoolName = src.Spec.PoolName
	dst.Spec.ServiceType = corev1.ServiceTypeClusterIP

	// Transform status
	dst.Status.Ready = src.Status.Ready
	dst.Status.Initialized = src.Status.Initialized
	dst.Status.ServiceReadyReplicas = src.Status.ReadyComponentNum
	dst.Status.ServiceReplicas = src.Status.ReadyComponentNum + src.Status.UnreadyComponentNum
	dst.Status.DeploymentReadyReplicas = src.Status.ReadyComponentNum
	dst.Status.DeploymentReplicas = src.Status.ReadyComponentNum + src.Status.UnreadyComponentNum
	dst.Status.Conditions = src.Status.Conditions

	//TODO: AdditionalService and AdditionalDeployment
	return nil
}
