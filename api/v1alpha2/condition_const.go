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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// ConfigmapAvailableCondition documents the status of the EdgeX configmap.
	ConfigmapAvailableCondition clusterv1.ConditionType = "ConfigmapAvailable"

	ConfigmapProvisioningReason = "ConfigmapProvisioning"

	ConfigmapProvisioningFailedReason = "ConfigmapProvisioningFailed"
	// ComponentAvailableCondition documents the status of the EdgeX component.
	ComponentAvailableCondition clusterv1.ConditionType = "ComponentAvailable"

	ComponentProvisioningReason = "ComponentProvisioning"

	ComponentProvisioningFailedReason = "ComponentProvisioningFailed"
)
