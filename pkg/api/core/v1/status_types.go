package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionType is a valid value of Condition.Type
type ConditionType string

const (
	// ConditionReady indicates the resource is ready.
	ConditionReady ConditionType = "Ready"
)

// Judgment contains details about resource state
type Judgment struct {
	// Condition type.
	// +required
	Type ConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`
	// Condition status, True, False or Unknown.
	// +required
	Status corev1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`
	// Last time the condition transitioned from one status to another.
	// +optional
	EventId            string      `json:"eventId" protobuf:"bytes,3,opt,name=eventId"`
	TriggerId          string      `json:"triggerId" protobuf:"bytes,4,opt,name=triggerId"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,5,opt,name=lastTransitionTime"`
	// Unique, this should be a short, machine understandable string that gives the reason
	// for condition's last transition. For example, "ImageNotFound"
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,6,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,7,opt,name=message"`
}

// Status is a common structure which can be used for Status field.
type Status struct {
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Judgment *Judgment `json:"judgment,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=judgment"`
}

// InitializeConditions initializes the contions to Unknown
func (s *Status) InitializeConditions(judge *Judgment) {
	s.Judgment = judge
}

// IsReady returns true when all the conditions are true
func (s *Status) IsReady() bool {
	if s.Judgment.EventId != "" && s.Judgment.TriggerId != "" && s.Judgment.Status == corev1.ConditionTrue {
		return true
	}
	return false

}
