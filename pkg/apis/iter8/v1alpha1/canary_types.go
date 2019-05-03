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

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
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

	// Analysis parameters
	Analysis Analysis `json:"analysis"`
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

	// AnalysisState is the last analysis state
	AnalysisState runtime.RawExtension `json:"analysisState"`
}

type TrafficControl struct {
	Strategy string `json:"strategy"`

	// MaxTrafficPercent is the maximum traffic ratio to send to the canary. Default is 50
	// +optional
	// Deprecated
	MaxTrafficPercent *int `json:"maxTrafficPercent,omitempty"`

	// StepSize is the traffic increment per interval. Default is 2
	// +optional
	// Deprecated
	StepSize *int `json:"stepSize,omitempty"`

	// Interval is the time in second before the next increment. Default is 1mn
	// +optional
	Interval *time.Duration `json:"interval,omitempty"`
}

type Analysis struct {
	// AnalyticsService endpoint
	AnalyticsService string `json:"analyticsService"`

	// List of criteria for assessing the canary version
	Metrics []SuccessCriterion `json:"metrics"`
}

type SuccessCriterion struct {
	// Name of the metric to which the criterion applies
	// example: iter8_latency
	Name string `json:"name"`

	// 	Criterion type. Options:
	// "delta": compares the canary against the baseline version with respect to the metric;
	// "threshold": checks the canary with respect to the metric
	Type checkandincrement.SuccessCriterionType `json:"type"`

	// Determines how the traffic must be split at the end of the experiment; options:
	// "baseline": all traffic goes to the baseline version;
	// "canary": all traffic goes to the canary version;
	// "both": traffic is split across baseline and canary.
	// Defaults to “canary”
	OnSuccess string `json:"onSuccess"`

	// Value to check
	Value float64 `json:"value"`

	// Minimum number of data points required to make a decision based on this criterion;
	// if not specified, there is no requirement on the sample size
	SampleSize int `json:"sampleSize"`
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

	// CanaryConditionRolloutCompleted has status True when the Canary rollout is completed
	CanaryConditionRolloutCompleted duckv1alpha1.ConditionType = "RolloutCompleted"
)

var canaryCondSet = duckv1alpha1.NewLivingConditionSet(
	CanaryConditionServiceProvided,
	CanaryConditionRolloutCompleted,
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

// MarkHasService sets the condition that the target service has been found
func (s *CanaryStatus) MarkRolloutCompleted() {
	canaryCondSet.Manage(s).MarkTrue(CanaryConditionRolloutCompleted)
}

// MarkHasNotService sets the condition that the target service hasn't been found.
func (s *CanaryStatus) MarkRolloutNotCompleted(reason, messageFormat string, messageA ...interface{}) {
	canaryCondSet.Manage(s).MarkFalse(CanaryConditionRolloutCompleted, reason, messageFormat, messageA...)
}

func init() {
	SchemeBuilder.Register(&Canary{}, &CanaryList{})
}
