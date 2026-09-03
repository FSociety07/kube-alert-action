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

// TODO
type AlertEventStatus struct {
}
