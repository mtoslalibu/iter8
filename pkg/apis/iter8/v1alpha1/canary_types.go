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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Canary is the Schema for the canaries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:categories=all,iter8
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

	// LastIncrementTime is the last time the traffic has been incremented
	LastIncrementTime metav1.Time `json:"lastIncrementTime"`

	// Progressing is true when rollout is in progress
	Progressing bool `json:"progressing"`
}

type TrafficControl struct {
	Strategy string `json:"strategy"`

	// MaxTrafficPercent is the maximum traffic ratio to send to the canary. Default is 50
	// +optional
	MaxTrafficPercent *int `json:"maxTrafficPercent,omitempty"`

	// StepSize is the traffic increment per interval. Default is 2
	// +optional
	StepSize *int `json:"stepSize,omitempty"`

	// Interval is the time in second before the next increment. Default is 1mn
	// +optional
	Interval *time.Duration `json:"interval,omitempty"`
}

// GetMaxTrafficPercent gets the specified max traffic percent or the default value (50)
func (t *TrafficControl) GetMaxTrafficPercent() int {
	maxPercent := t.MaxTrafficPercent
	if maxPercent == nil {
		fifty := int(50)
		maxPercent = &fifty
	}
	return *maxPercent
}

// GetStepSize gets the specified step size or the default value (2%)
func (t *TrafficControl) GetStepSize() int {
	stepSize := t.StepSize
	if stepSize == nil {
		two := int(2)
		stepSize = &two
	}
	return *stepSize
}

// GetInterval gets the specified interval or the default value (1mn)
func (t *TrafficControl) GetInterval() time.Duration {
	interval := t.Interval
	if interval == nil {
		onemn := time.Minute
		interval = &onemn
	}
	return *interval
}

const (
	// CanaryConditionReady has status True when the Canary has finished controlling traffic
	CanaryConditionReady = duckv1alpha1.ConditionReady

	// CanaryConditionServiceProvided has status True when the Canary has been configured with a Knative service
	CanaryConditionServiceProvided duckv1alpha1.ConditionType = "ServiceProvided"

	// CanaryConditionMinimumRevisionsAvailable has status True when the Knative service has at least two revisions
	CanaryConditionMinimumRevisionsAvailable duckv1alpha1.ConditionType = "MinimumRevisionsAvailable"
)

var canaryCondSet = duckv1alpha1.NewLivingConditionSet(
	CanaryConditionServiceProvided,
	CanaryConditionMinimumRevisionsAvailable,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *CanaryStatus) InitializeConditions() {
	canaryCondSet.Manage(s).InitializeConditions()
}

// MarkHasService sets the condition that the target service has been found
func (s *CanaryStatus) MarkHasService() {
	canaryCondSet.Manage(s).MarkTrue(CanaryConditionServiceProvided)
}

// MarkHasNotService sets the condition that the target service hasn't been found.
func (s *CanaryStatus) MarkHasNotService(reason, messageFormat string, messageA ...interface{}) {
	canaryCondSet.Manage(s).MarkFalse(CanaryConditionServiceProvided, reason, messageFormat, messageA...)
}

// MarkMinimumRevisionAvailable sets the condition that the target service has enough revisions
func (s *CanaryStatus) MarkMinimumRevisionAvailable() {
	canaryCondSet.Manage(s).MarkTrue(CanaryConditionMinimumRevisionsAvailable)
}

// MarkMinimumRevisionNotAvailable sets the condition thatthe target service has not enough revisions
func (s *CanaryStatus) MarkMinimumRevisionNotAvailable(reason, messageFormat string, messageA ...interface{}) {
	canaryCondSet.Manage(s).MarkFalse(CanaryConditionMinimumRevisionsAvailable, reason, messageFormat, messageA...)
}

func init() {
	SchemeBuilder.Register(&Canary{}, &CanaryList{})
}
