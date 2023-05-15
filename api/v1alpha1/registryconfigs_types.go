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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RegistryConfigsSpec defines the desired state of RegistryConfigs.
type RegistryConfigsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Selector is used to select nodes that will be configured
	Selector metav1.LabelSelector    `json:"selector,omitempty"`
	Template RegistryConfigsTemplate `json:"template,omitempty"`
}

// RegistryConfigsTemplate defines the template for the registry config.
type RegistryConfigsTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeHostConfigsSpec `json:"spec"`
}

// NodeHostConfigsSpec defines the host config for the registry.
type NodeHostConfigsSpec struct {
	RetryNum    int                      `json:"retry_num,omitempty"`
	HostConfigs []NodeRegistryHostConfig `json:"hostConfigs"`
}

// RegistryConfigsStatus defines the observed state of RegistryConfigs.
type RegistryConfigsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State        StatusState  `json:"state,omitempty"`
	TotalNodes   []RuntimeNum `json:"total_nodes,omitempty"`
	SuccessNodes []RuntimeNum `json:"success_nodes,omitempty"`
	FailedNodes  []RuntimeNum `json:"failed_nodes,omitempty"`
	RunningNodes []RuntimeNum `json:"running_nodes,omitempty"`
}

type RuntimeNum struct {
	RuntimeType RuntimeType `json:"runtimeType,omitempty"`
	Num         int         `json:"num,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// RegistryConfigs is the Schema for the registryconfigs API.
type RegistryConfigs struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistryConfigsSpec   `json:"spec,omitempty"`
	Status RegistryConfigsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RegistryConfigsList contains a list of RegistryConfigs.
type RegistryConfigsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegistryConfigs `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegistryConfigs{}, &RegistryConfigsList{})
}
