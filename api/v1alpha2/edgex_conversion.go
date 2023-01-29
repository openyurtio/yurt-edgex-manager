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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
)

func (src *EdgeX) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.EdgeX)
	dst.ObjectMeta = src.ObjectMeta
	dst.TypeMeta = src.TypeMeta
	dst.TypeMeta.APIVersion = "device.openyurt.io/v1alpha1"
	dst.Spec.Version = src.Spec.Version
	dst.Spec.ImageRegistry = src.Spec.ImageRegistry
	dst.Spec.PoolName = src.Spec.PoolName
	dst.Spec.ServiceType = corev1.ServiceTypeClusterIP
	//TODO: AdditionalService and AdditionalDeployment

	dst.Status.Ready = src.Status.Ready
	dst.Status.Initialized = src.Status.Initialized
	dst.Status.ServiceReadyReplicas = src.Status.ReadyComponentNum
	dst.Status.ServiceReplicas = src.Status.ReadyComponentNum + src.Status.UnreadyComponentNum
	dst.Status.DeploymentReadyReplicas = src.Status.ReadyComponentNum
	dst.Status.DeploymentReplicas = src.Status.ReadyComponentNum + src.Status.UnreadyComponentNum
	dst.Status.Conditions = src.Status.Conditions

	return nil
}
func (dst *EdgeX) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.EdgeX)
	dst.ObjectMeta = src.ObjectMeta
	dst.TypeMeta = src.TypeMeta
	dst.TypeMeta.APIVersion = "device.openyurt.io/v1alpha2"
	dst.Spec.Version = src.Spec.Version
	dst.Spec.Security = false
	dst.Spec.ImageRegistry = src.Spec.ImageRegistry
	dst.Spec.PoolName = src.Spec.PoolName
	//TODO: Components
	dst.Status.Ready = src.Status.Ready
	dst.Status.Initialized = src.Status.Initialized
	dst.Status.ReadyComponentNum = src.Status.DeploymentReadyReplicas
	dst.Status.UnreadyComponentNum = src.Status.DeploymentReplicas - src.Status.DeploymentReadyReplicas
	dst.Status.Conditions = src.Status.Conditions
	return nil
}
