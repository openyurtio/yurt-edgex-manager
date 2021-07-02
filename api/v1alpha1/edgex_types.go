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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentTemplateSpec defines the pool template of Deployment.
type DeploymentTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              appsv1.DeploymentSpec `json:"spec"`
}

// DeploymentTemplateSpec defines the pool template of Deployment.
type ServiceTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              appsv1.DeploymentSpec `json:"spec"`
}

type ComponetSpec struct {
	// +optional
	Deployment DeploymentTemplateSpec `json:"deploymentspec,omitempty"`

	// +optional
	Service ServiceTemplateSpec `json:"servicespec,omitempty"`
}

// EdgeXSpec defines the desired state of EdgeX
type EdgeXSpec struct {
	Version string `json:"version,omitempty"`

	PoolName string `json:"poolname,omitempty"`

	AdditionalComponents []ComponetSpec `json:"additinalcomponets,omitempty"`
}

type ComponetStatus struct {
	Deployment appsv1.DeploymentStatus `json:"deploymentstatus,omitempty"`
}

// EdgeXStatus defines the observed state of EdgeX
type EdgeXStatus struct {
	Initialized bool `json:"initialized,omitempty"`

	ComponetStatus []ComponetStatus `json:"componetsstatus,omitempty"`

	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=edgexes

// EdgeX is the Schema for the edgexes API
type EdgeX struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeXSpec   `json:"spec,omitempty"`
	Status EdgeXStatus `json:"status,omitempty"`
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
