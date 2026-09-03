package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
type ActionMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionMapSpec   `json:"spec,omitempty"`
	Status ActionMapStatus `json:"status,omitempty"`
}

type ActionMapSpec struct {
	Rules []ActionRule `json:"rules"`
}

type ActionRule struct {
	Metric string `json:"metric"`
	Action string `json:"action"`
	// +kubebuilder:validation:Enum=self;targetPod
	ExecuteFrom string `json:"executeFrom"`
}

// TODO
type ActionMapStatus struct {
}

// +kubebuilder:object:root=true
type ActionMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ActionMap `json:"items"`
}
