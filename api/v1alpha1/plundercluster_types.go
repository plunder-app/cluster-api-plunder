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
	// ClusterFinalizer allows Reconciler to clean up resources associated with PlunderCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "plundercluster.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlunderClusterSpec defines the desired state of PlunderCluster
type PlunderClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// StaticMAC denotes that the machine is ready
	StaticMAC string `json:"staticMAC,omitempty"`

	// StaticIp denotes that the machine is ready
	StaticIp string `json:"staticIP,omitempty"`
}

// PlunderClusterStatus defines the observed state of PlunderCluster
type PlunderClusterStatus struct {
	// Ready denotes that the machine is ready
	Ready bool `json:"ready"`

	// APIEndpoints represents the endpoints to communicate with the control plane.
	// +optional
	APIEndpoints []APIEndpoint `json:"apiEndpoints,omitempty"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {
	// Host is the hostname on which the API server is serving.
	Host string `json:"host"`

	// Port is the port on which the API server is serving.
	Port int `json:"port"`
}

// +kubebuilder:object:root=true

// PlunderCluster is the Schema for the plunderclusters API
type PlunderCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlunderClusterSpec   `json:"spec,omitempty"`
	Status PlunderClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlunderClusterList contains a list of PlunderCluster
type PlunderClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlunderCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlunderCluster{}, &PlunderClusterList{})
}
