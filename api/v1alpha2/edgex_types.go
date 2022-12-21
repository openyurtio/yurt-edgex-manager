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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// name of finalizer
	EdgexFinalizer = "edgex.edgexfoundry.org"

	LabelEdgeXGenerate = "www.edgexfoundry.org/generate"
)

type Component struct {
	Name string `json:"name"`

	// +optional
	Image string `json:"image,omitempty"`

	// Replicas int32 `json:"replicas,omitempty"`
}

// EdgeXSpec defines the desired state of EdgeX
type EdgeXSpec struct {
	Version string `json:"version,omitempty"`

	ImageRegistry string `json:"imageRegistry,omitempty"`

	PoolName string `json:"poolName,omitempty"`

	// +optional
	Components []Component `json:"components,omitempty"`

	// +optional
	Security bool `json:"security,omitempty"`
}

// EdgeXStatus defines the observed state of EdgeX
type EdgeXStatus struct {
	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Initialized bool `json:"initialized,omitempty"`

	// +optional
	ReadyComponentNum int32 `json:"readyComponentNum,omitempty"`

	// +optional
	UnreadyComponentNum int32 `json:"unreadyComponentNum,omitempty"`

	// Current Edgex state
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=edgexes
//+kubebuilder:resource:shortName=edgex
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="The edgex ready status"
//+kubebuilder:printcolumn:name="ReadyComponentNum",type="integer",JSONPath=".status.readyComponentNum",description="The Ready Component."
//+kubebuilder:printcolumn:name="UnreadyComponentNum",type="integer",JSONPath=".status.unreadyComponentNum",description="The Unready Component."
//+kubebuilder:unservedversion

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
