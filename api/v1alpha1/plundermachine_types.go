/*
Copyright 2019 The Kubernetes Authors.

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

const (
	// MachineFinalizer allows the reconciler to clean up resources associated with PlunderMachine before
	// removing it from the apiserver.
	MachineFinalizer = "plundernmachine.infrastructure.cluster.x-k8s.io"

	// DockerVersionDefault is the version of Docker that the provider will default to
	DockerVersionDefault = "18.06.1~ce~3-0~ubuntu"

	// KubernetesVersionDefault is the version of Docker that the provider will default to
	KubernetesVersionDefault = "v1.15.1"

	// DeploymentDefault is the default type of installation
	DeploymentDefault = "preseed"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlunderMachineSpec defines the desired state of PlunderMachine
type PlunderMachineSpec struct {
	// ProviderID will be the only detail (todo: something else)
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// ControlPlaneMac will be a pool of mac addresses for control plane nodes
	// +optional
	ControlPlaneMacPool []string `json:"controlPlaneMacPool,omitempty"`

	// MACHINE Definition

	// IPAddress is the address to be used IF IPAM isn't enabled (SPOILER IT ISN'T as i've not written it yet)
	IPAddress *string `json:"ipaddress,omitempty"`

	// MACAddress is the physical network address of the if we don't auto detect
	MACAddress *string `json:"macaddress,omitempty"`

	// DockerVersion is the version of the docker engine that will be installed
	DockerVersion *string `json:"dockerVersion,omitempty"`

	// DeploymentType defines what will be deployed on the new machine
	// +optional
	DeploymentType *string `json:"deploymentType,omitempty"`
}

// PlunderMachineStatus defines the observed state of PlunderMachine
type PlunderMachineStatus struct {
	// Ready denotes that the machine is ready
	Ready bool `json:"ready"`

	// IPAdress is the allocated networking address
	IPAdress string `json:"ipaddress,omitempty"`
	// MACAddress is the physical network address of the machine
	MACAddress string `json:"macaddress,omitempty"`
}

// +kubebuilder:object:root=true

// PlunderMachine is the Schema for the plundermachines API
type PlunderMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlunderMachineSpec   `json:"spec,omitempty"`
	Status PlunderMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlunderMachineList contains a list of PlunderMachine
type PlunderMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlunderMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlunderMachine{}, &PlunderMachineList{})
}
