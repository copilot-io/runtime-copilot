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

type (
	// RuntimeType is the type of runtime.
	RuntimeType string
	// CapabilitieType is the type of capability.
	CapabilitieType string
	// StatusState is the state of status.
	StatusState string
	// ConditionType is the type of condition.
	ConditionType string
)

const (
	RuntimeTypeDocker     RuntimeType = "docker"
	RuntimeTypeContainerd RuntimeType = "containerd"
	RuntimeTypeCrio       RuntimeType = "crio"
	RuntimeTypeUnknown    RuntimeType = "unknown"

	CapabilitieTypePull    CapabilitieType = "pull"
	CapabilitieTypePush    CapabilitieType = "push"
	CapabilitieTypeResolve CapabilitieType = "resolve"

	StatusStateSuccess StatusState = "Success"
	StatusStateFailed  StatusState = "Failed"
	StatusStateRunning StatusState = "Running"
	StatusStateUnknown StatusState = "Unknown"

	ConditionTypeWriteDataError ConditionType = "WriteDataError"
	ConditionTypeReadDataError  ConditionType = "ReadDataError"

	// MaxRetryNum is the max retry num.
	MaxRetryNum = 3
)

func (rt RuntimeType) String() string {
	return string(rt)
}

func (ct CapabilitieType) String() string {
	return string(ct)
}

func (ss StatusState) String() string {
	return string(ss)
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeRegistryConfigsSpec defines the desired state of NodeRegistryConfigs.
type NodeRegistryConfigsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NodeName is registry config exec node name
	NodeName string `json:"nodeName"`
	// Type is runtime type
	Type RuntimeType `json:"type"`
	// set retry num
	RetryNum int `json:"retry_num,omitempty"`
	// HostConfigs store the per-host configuration
	HostConfigs []NodeRegistryHostConfig `json:"hostConfigs,omitempty"`
}

type NodeRegistryHostConfig struct {
	// Server specifies the default server. When `host` is
	// also specified, those hosts are tried first.
	Server       string            `json:"server,omitempty"`
	Capabilities []CapabilitieType `json:"capabilities"`
	SkipVerify   *bool             `json:"skip_verify,omitempty"`
	Header       map[string]string `json:"header,omitempty"`
	OverridePath bool              `json:"override_path,omitempty"`
	CaSecretRef  string            `json:"ca_secret_ref,omitempty"`
}

// NodeRegistryConfigsStatus defines the observed state of NodeRegistryConfigs.
type NodeRegistryConfigsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	State      StatusState        `json:"state,omitempty"`
	RetryNum   int                `json:"retry_num,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// NodeRegistryConfigs is the Schema for the noderegistryconfigs API.
type NodeRegistryConfigs struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeRegistryConfigsSpec   `json:"spec,omitempty"`
	Status NodeRegistryConfigsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeRegistryConfigsList contains a list of NodeRegistryConfigs.
type NodeRegistryConfigsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeRegistryConfigs `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeRegistryConfigs{}, &NodeRegistryConfigsList{})
}
