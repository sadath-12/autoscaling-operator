/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CustomAutoScalingSpec defines the desired state of CustomAutoScaling
type CustomAutoScalingSpec struct {
	ApplicationRef       ApplicationReference `json:"applicationRef"`
	ScalingParamsMapping map[string]string    `json:"scalingParamsMapping"`
	ScalingQuery         string               `json:"scalingQuery"`
} 

// ApplicationReference defines the deployment to scale
type ApplicationReference struct {
	DeploymentName string `json:"deploymentName"`
}

// CustomAutoScalingStatus defines the observed state of CustomAutoScaling
type CustomAutoScalingStatus struct {
	Replicas int32 `json:"replicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CustomAutoScaling is the Schema for the customautoscalings API
type CustomAutoScaling struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomAutoScalingSpec   `json:"spec,omitempty"`
	Status CustomAutoScalingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CustomAutoScalingList contains a list of CustomAutoScaling
type CustomAutoScalingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomAutoScaling `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomAutoScaling{}, &CustomAutoScalingList{})
}
