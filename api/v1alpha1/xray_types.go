/*
Copyright 2025.

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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// XraySpec defines the desired state of Xray.
type XraySpec struct {
	// ttlSecondsAfterFinished limits the lifetime of a Job that has finished
	// execution (either Complete or Failed). If this field is set,
	// ttlSecondsAfterFinished after the Job finishes, it is eligible to be
	// automatically deleted. When the Job is being deleted, its lifecycle
	// guarantees (e.g. finalizers) will be honored. If this field is unset,
	// the Job won't be automatically deleted. If this field is set to zero,
	// the Job becomes eligible to be deleted immediately after it finishes.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty" protobuf:"varint,8,opt,name=ttlSecondsAfterFinished"`

	// JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"` // We might not need this
	Template corev1.PodTemplateSpec `json:"template" protobuf:"bytes,6,opt,name=template"`

	// The only allowed template.spec.restartPolicy values are "Never" or "OnFailure".
	// Template corev1.PodTemplateSpec `json:"template" protobuf:"bytes,6,opt,name=template"`
}

// XrayStatus defines the observed state of Xray.
type XrayStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=atomic
	Conditions []batchv1.JobCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Represents time when the job controller started processing a job. When a
	// Job is created in the suspended state, this field is not set until the
	// first time it is resumed. This field is reset every time a Job is resumed
	// from suspension. It is represented in RFC3339 form and is in UTC.
	//
	// Once set, the field can only be removed when the job is suspended.
	// The field cannot be modified while the job is unsuspended or finished.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,2,opt,name=startTime"`

	// Represents time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	// The completion time is set when the job finishes successfully, and only then.
	// The value cannot be updated or removed. The value indicates the same or
	// later point in time as the startTime field.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty" protobuf:"bytes,3,opt,name=completionTime"`

	// The number of pending and running pods which are not terminating (without
	// a deletionTimestamp).
	// The value is zero for finished jobs.
	// +optional
	Active int32 `json:"active,omitempty" protobuf:"varint,4,opt,name=active"`

	// The number of pods which reached phase Succeeded.
	// The value increases monotonically for a given spec. However, it may
	// decrease in reaction to scale down of elastic indexed jobs.
	// +optional
	Succeeded int32 `json:"succeeded,omitempty" protobuf:"varint,5,opt,name=succeeded"`

	// The number of pods which reached phase Failed.
	// The value increases monotonically.
	// +optional
	Failed int32 `json:"failed,omitempty" protobuf:"varint,6,opt,name=failed"`

	// uncountedTerminatedPods holds the UIDs of Pods that have terminated but
	// the job controller hasn't yet accounted for in the status counters.
	//
	// The job controller creates pods with a finalizer. When a pod terminates
	// (succeeded or failed), the controller does three steps to account for it
	// in the job status:
	//
	// 1. Add the pod UID to the arrays in this field.
	// 2. Remove the pod finalizer.
	// 3. Remove the pod UID from the arrays while increasing the corresponding
	//     counter.
	//
	// Old jobs might not be tracked using this field, in which case the field
	// remains null.
	// The structure is empty for finished jobs.
	// +optional
	UncountedTerminatedPods *UncountedTerminatedPods `json:"uncountedTerminatedPods,omitempty" protobuf:"bytes,8,opt,name=uncountedTerminatedPods"`

	// The number of active pods which have a Ready condition and are not
	// terminating (without a deletionTimestamp).
	Ready *int32 `json:"ready,omitempty" protobuf:"varint,9,opt,name=ready"`
}

// UncountedTerminatedPods holds UIDs of Pods that have terminated but haven't
// been accounted in Job status counters.
type UncountedTerminatedPods struct {
	// succeeded holds UIDs of succeeded Pods.
	// +listType=set
	// +optional
	Succeeded []types.UID `json:"succeeded,omitempty" protobuf:"bytes,1,rep,name=succeeded,casttype=k8s.io/apimachinery/pkg/types.UID"`

	// failed holds UIDs of failed Pods.
	// +listType=set
	// +optional
	Failed []types.UID `json:"failed,omitempty" protobuf:"bytes,2,rep,name=failed,casttype=k8s.io/apimachinery/pkg/types.UID"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Xray is the Schema for the xrays API.
type Xray struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XraySpec   `json:"spec,omitempty"`
	Status XrayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// XrayList contains a list of Xray.
type XrayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Xray `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Xray{}, &XrayList{})
}
