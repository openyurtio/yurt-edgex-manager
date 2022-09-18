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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// name of finalizer
	EdgexFinalizer = "edgex.edgexfoundry.org"

	LabelEdgeXGenerate = "www.edgexfoundry.org/generate"
)

// DeploymentTemplateSpec defines the pool template of Deployment.
type DeploymentTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              appsv1.DeploymentSpec `json:"spec"`
}

// DeploymentTemplateSpec defines the pool template of Deployment.
type ServiceTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              corev1.ServiceSpec `json:"spec"`
}

// EdgeXSpec defines the desired state of EdgeX
type EdgeXSpec struct {
	Version string `json:"version,omitempty"`

	ImageRegistry string `json:"imageRegistry,omitempty"`

	PoolName string `json:"poolName,omitempty"`

	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
	// +optional
	AdditionalService []ServiceTemplateSpec `json:"additionalServices,omitempty"`

	// +optional
	AdditionalDeployment []DeploymentTemplateSpec `json:"additionalDeployments,omitempty"`
}

// EdgeXStatus defines the observed state of EdgeX
type EdgeXStatus struct {
	// +optional
	Ready bool `json:"ready,omitempty"`
	// +optional
	Initialized bool `json:"initialized,omitempty"`
	// +optional
	ServiceReplicas int32 `json:"serviceReplicas,omitempty"`
	// +optional
	ServiceReadyReplicas int32 `json:"serviceReadyReplicas,omitempty"`
	// +optional
	DeploymentReplicas int32 `json:"deploymentReplicas,omitempty"`
	// +optional
	DeploymentReadyReplicas int32 `json:"deploymentReadyReplicas,omitempty"`

	// Current Edgex state
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=edgexes
//+kubebuilder:resource:shortName=edgex
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="The edgex ready status"
//+kubebuilder:printcolumn:name="Service",type="integer",JSONPath=".status.servicereplicas",description="The Service Replica."
//+kubebuilder:printcolumn:name="ReadyService",type="integer",JSONPath=".status.servicereadyreplicas",description="The Ready Service Replica."
//+kubebuilder:printcolumn:name="Deployment",type="integer",JSONPath=".status.deploymentreplicas",description="The Deployment Replica."
//+kubebuilder:printcolumn:name="ReadyDeployment",type="integer",JSONPath=".status.deploymentreadyreplicas",description="The Ready Deployment Replica."

// EdgeX is the Schema for the edgexes API
type EdgeX struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeXSpec   `json:"spec,omitempty"`
	Status EdgeXStatus `json:"status,omitempty"`
}

func (c *EdgeX) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *EdgeX) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// EdgeXList contains a list of EdgeX
type EdgeXList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeX `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeX{}, &EdgeXList{})
}
