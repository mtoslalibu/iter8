/*

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Canary is the Schema for the canaries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Canary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CanarySpec   `json:"spec,omitempty"`
	Status CanaryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// CanaryList contains a list of Canary
type CanaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Canary `json:"items"`
}

// CanarySpec defines the desired state of Canary
type CanarySpec struct {
	// TargetService is a reference to an object to use as target service
	TargetService *corev1.ObjectReference `json:"targetService"`

	// TrafficControl defines parameters for controlling the traffic
	TrafficControl TrafficControl `json:"trafficControl"`
}

// CanaryStatus defines the observed state of Canary
type CanaryStatus struct {
	// inherits duck/v1alpha1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1alpha1.Status `json:",inline"`
}

type TrafficControl struct {
	Strategy string `json:"strategy"`

	// MaxTrafficPercent is the maximum traffic ratio to send to the canary. Default is 50
	// +optional
	MaxTrafficPercent *float32 `json:"maxTrafficPercent,omitempty"`

	// StepSize is the traffic increment per interval. Default is 2
	// +optional
	StepSize *float32 `json:"stepSize,omitempty"`

	// Interval is the time in second before the next increment. Default is 1mn
	// +optional
	Interval *int32
}

const (
	// CanaryConditionReady has status True when the Canary is ready to control traffic
	CanaryConditionReady = duckv1alpha1.ConditionReady

	// CanaryConditionServiceProvided has status True when the Canary has been configured with a Knative service
	CanaryConditionServiceProvided duckv1alpha1.ConditionType = "ServiceProvided"
)

var canaryCondSet = duckv1alpha1.NewLivingConditionSet(
	CanaryConditionServiceProvided,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *CanaryStatus) InitializeConditions() {
	canaryCondSet.Manage(s).InitializeConditions()
}

// MarkHasService sets the condition that the target service has been found and configured.
func (s *CanaryStatus) MarkHasService() {
	canaryCondSet.Manage(s).MarkTrue(CanaryConditionServiceProvided)
}

// MarkHasNotService sets the condition that the target service hasn't been found.
func (s *CanaryStatus) MarkHasNotService(reason, messageFormat string, messageA ...interface{}) {
	canaryCondSet.Manage(s).MarkFalse(CanaryConditionServiceProvided, reason, messageFormat, messageA...)
}

func init() {
	SchemeBuilder.Register(&Canary{}, &CanaryList{})
}
