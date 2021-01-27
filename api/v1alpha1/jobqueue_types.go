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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobQueueSpec defines the desired state of JobQueue
type JobQueueSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Work is name of JobQueue.
	Work string `json:"work,omitempty"`

	// Size is the max size of the deployment
	Size int32 `json:"size"`

	// Work Size is the size of the work
	WorkSize int32 `json:"worksize"`
}

// JobQueueStatus defines the observed state of JobQueue
type JobQueueStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Nodes are the names of the JobQueue pods
	Nodes []string `json:"nodes"`

	// Status of the JobQueue
	Status string `json:"status"`

	// Amount of work left in the queue
	WorkQueueSize int32 `json:"workqueuesize"`

	// Number of active jobs (<= Deployment Size)
	ActiveJobs int32 `json:"activejobs"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JobQueue is the Schema for the jobqueues API
type JobQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobQueueSpec   `json:"spec,omitempty"`
	Status JobQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobQueueList contains a list of JobQueue
type JobQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobQueue{}, &JobQueueList{})
}
