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
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Canary is the Schema for the canaries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:categories=all,iter8
// +kubebuilder:printcolumn:name="status",type="string",JSONPath=".status.conditions[?(@.type == 'Ready')].reason",description="Status of the experiment",format="byte"
// +kubebuilder:printcolumn:name="baseline",type="string",JSONPath=".spec.targetService.baseline",description="Name of baseline",format="byte"
// +kubebuilder:printcolumn:name="percentage",type="integer",JSONPath=".status.trafficSplitPercentage.baseline",description="Traffic percentage for baseline",format="int32"
// +kubebuilder:printcolumn:name="canary",type="string",JSONPath=".spec.targetService.candidate",description="Name of canary",format="byte"
// +kubebuilder:printcolumn:name="percentage",type="integer",JSONPath=".status.trafficSplitPercentage.canary",description="Traffic percentage for canary",format="int32"
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
	TargetService TargetService `json:"targetService"`

	// TrafficControl defines parameters for controlling the traffic
	TrafficControl TrafficControl `json:"trafficControl"`

	// Analysis parameters
	// +optional
	Analysis Analysis `json:"analysis,omitempty"`
}

// TargetService defines what to watch in the controller
type TargetService struct {
	// defines the characteristics of the service
	*corev1.ObjectReference `json:",inline"`

	// Baseline tells the name of baseline
	Baseline string `json:"baseline,omitempty"`

	// Candidate tells the name of candidate
	Candidate string `json:"candidate,omitempty"`
}

// CanaryStatus defines the observed state of Canary
type CanaryStatus struct {
	// inherits duck/v1alpha1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1alpha1.Status `json:",inline"`

	// LastIncrementTime is the last time the traffic has been incremented
	LastIncrementTime metav1.Time `json:"lastIncrementTime,omitempty"`

	// CurrentIteration is the current iteration number
	CurrentIteration int `json:"currentIteration,omitempty"`

	// AnalysisState is the last analysis state
	AnalysisState runtime.RawExtension `json:"analysisState,omitempty"`

	// AssessmentSummary returned by the last analyis
	AssessmentSummary Summary `json:"assessment,omitempty"`

	// TrafficSplit tells the current traffic spliting between baseline and canary
	TrafficSplit TrafficSplit `json:"trafficSplitPercentage,omitempty"`
}

type TrafficSplit struct {
	Baseline int `json:"baseline"`
	Canary   int `json:"canary"`
}

type TrafficControl struct {
	Strategy string `json:"strategy"`

	// MaxTrafficPercent is the maximum traffic ratio to send to the canary. Default is 50
	// +optional
	MaxTrafficPercent *float64 `json:"maxTrafficPercent,omitempty"`

	// StepSize is the traffic increment per interval. Default is 2
	// +optional
	StepSize *float64 `json:"stepSize,omitempty"`

	// Interval is the time in second before the next increment. Default is 1mn
	// +optional
	Interval *string `json:"interval,omitempty"`

	// Number of iterations for this experiment. Default to 100.
	// +optional
	IterationCount *int `json:"iterationCount,omitempty"`

	// Determines how the traffic must be split at the end of the experiment; options:
	// "baseline": all traffic goes to the baseline version;
	// "canary": all traffic goes to the canary version;
	// "both": traffic is split across baseline and canary.
	// Defaults to “canary”
	// +optional
	//+kubebuilder:validation:Enum=baseline,canary,both
	OnSuccess *string `json:"onSuccess,omitempty"`
}

type Analysis struct {
	// AnalyticsService endpoint
	AnalyticsService string `json:"analyticsService"`

	// List of criteria for assessing the canary version
	Metrics []SuccessCriterion `json:"metrics"`
}

type Summary struct {
	// Overall summary based on all success criteria
	Conclusions []string `json:"conclusions,omitempty"`

	// Indicates whether or not all success criteria for assessing the canary version
	// have been met
	AllSuccessCriteriaMet bool `json:"all_success_criteria_met,omitempty"`

	// Indicates whether or not the experiment must be aborted based on the success criteria
	AbortExperiment bool `json:"abort_experiment,omitempty"`
}

type SuccessCriterionType string

const (
	SuccessCriterionDelta     SuccessCriterionType = "delta"
	SuccessCriterionThreshold SuccessCriterionType = "threshold"
)

type SuccessCriterion struct {
	// Name of the metric to which the criterion applies
	// example: iter8_latency
	Name string `json:"name"`

	// 	Criterion type. Options:
	// "delta": compares the canary against the baseline version with respect to the metric;
	// "threshold": checks the canary with respect to the metric
	Type SuccessCriterionType `json:"type"`

	// Value to check
	Value float64 `json:"value"`

	// Minimum number of data points required to make a decision based on this criterion;
	// if not specified, there is no requirement on the sample size
	// +optional
	SampleSize *int `json:"sampleSize,omitempty"`

	// Indicates whether or not the experiment must finish if this criterion is not satisfied;
	// defaults to false
	// +optional
	StopOnFailure *bool `json:"stopOnFailure,omitempty"`

	// Indicates whether or not this criterion is considered for traffic-control decisions;
	// defaults to true
	// +optional
	EnableTrafficControl *bool `json:"enableTrafficControl,omitempty"`

	// Indicates that this criterion is based on statistical confidence;
	// for instance, one can specify a 98% confidence that the criterion is satisfied;
	// if not specified, there is no confidence requirement
	// +optional
	Confidence *float64 `json:"confidence,omitempty"`
}

// GetMaxTrafficPercent gets the specified max traffic percent or the default value (50)
func (t *TrafficControl) GetMaxTrafficPercent() float64 {
	maxPercent := t.MaxTrafficPercent
	if maxPercent == nil {
		fifty := float64(50)
		maxPercent = &fifty
	}
	return *maxPercent
}

// GetStepSize gets the specified step size or the default value (2%)
func (t *TrafficControl) GetStepSize() float64 {
	stepSize := t.StepSize
	if stepSize == nil {
		two := float64(2)
		stepSize = &two
	}
	return *stepSize
}

// GetIterationCount gets the number of iterations or the default value (100)
func (t *TrafficControl) GetIterationCount() int {
	count := t.IterationCount
	if count == nil {
		hundred := int(100)
		count = &hundred
	}
	return *count
}

// GetInterval gets the specified interval or the default value (1m)
func (t *TrafficControl) GetInterval() string {
	interval := t.Interval
	if interval == nil {
		onemn := "1m"
		interval = &onemn
	}
	return *interval
}

// GetIntervalDuration gets the specified interval or the default value (1mn)
func (t *TrafficControl) GetIntervalDuration() (time.Duration, error) {
	interval := t.GetInterval()

	return time.ParseDuration(interval)
}

// Get how the traffic must be split at the end of the experiment; Default is "canary"
func (t *TrafficControl) GetOnSuccess() string {
	onsuccess := t.OnSuccess
	if onsuccess == nil {
		canary := "canary"
		onsuccess = &canary
	}
	return *onsuccess
}

const (
	// CanaryConditionReady has status True when the Canary has finished controlling traffic
	CanaryConditionReady = duckv1alpha1.ConditionReady

	// CanaryConditionServiceProvided has status True when the Canary has been configured with a Knative service
	CanaryConditionServiceProvided duckv1alpha1.ConditionType = "ServiceProvided"

	// CanaryConditionExperimentCompleted has status True when the Canary rollout is completed
	CanaryConditionExperimentCompleted duckv1alpha1.ConditionType = "ExperimentCompleted"

	// CanaryConditionRollForwardSucceeded has status True when a Canary rollout forward is completed
	CanaryConditionRollForwardSucceeded duckv1alpha1.ConditionType = "RollForwardSucceeded"
)

var canaryCondSet = duckv1alpha1.NewLivingConditionSet(
	CanaryConditionServiceProvided,
	CanaryConditionExperimentCompleted,
	CanaryConditionRollForwardSucceeded,
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

// MarkExperimentCompleted sets the condition that the experiemnt is completed
func (s *CanaryStatus) MarkExperimentCompleted() {
	canaryCondSet.Manage(s).MarkTrue(CanaryConditionExperimentCompleted)
}

// MarkExperimentNotCompleted sets the condition that the experiemnt is ongoing
func (s *CanaryStatus) MarkExperimentNotCompleted(reason, messageFormat string, messageA ...interface{}) {
	canaryCondSet.Manage(s).MarkFalse(CanaryConditionExperimentCompleted, reason, messageFormat, messageA...)
}

// MarkRollForwardSucceeded sets the condition that
func (s *CanaryStatus) MarkRollForwardSucceeded() {
	canaryCondSet.Manage(s).MarkTrue(CanaryConditionRollForwardSucceeded)
}

// MarkRollForwardNotSucceeded sets the condition that the experiemnt is ongoing
func (s *CanaryStatus) MarkRollForwardNotSucceeded(reason, messageFormat string, messageA ...interface{}) {
	canaryCondSet.Manage(s).MarkFalse(CanaryConditionRollForwardSucceeded, reason, messageFormat, messageA...)
}

func init() {
	SchemeBuilder.Register(&Canary{}, &CanaryList{})
}
