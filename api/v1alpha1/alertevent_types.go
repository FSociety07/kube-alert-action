package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
type AlertEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertEventSpec   `json:"spec,omitempty"`
	Status AlertEventStatus `json:"status,omitempty"`
}

type AlertEventSpec struct {
	TargetNamespace string `json:"targetNamespace"`
	Container       string `json:"container"`
	Pod             string `json:"pod"`
	Metric          string `json:"metric"`
	Action          string `json:"action"`
	ExecuteFrom     string `json:"executeFrom"`
}

type AlertEventStatus struct {
	Phase      string       `json:"phase,omitempty"`
	ExecutedAt *metav1.Time `json:"executedAt,omitempty"`
	Message    string       `json:"message,omitempty"`
}

const (
	PhasePending    = "Pending"
	PhaseInProgress = "InProgress"
	PhaseCompleted  = "Completed"
	PhaseFailed     = "Failed"
)

// +kubebuilder:object:root=true
type AlertEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AlertEvent `json:"items"`
}
