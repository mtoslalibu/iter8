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

package v1alpha2

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var experimentCondSet = []ExperimentConditionType{
	ExperimentConditionMetricsSynced,
	ExperimentConditionTargetsProvided,
	ExperimentConditionExperimentCompleted,
	ExperimentConditionAnalyticsServiceNormal,
	ExperimentConditionRoutingRulesReady,
}

func (s *ExperimentStatus) addCondition(conditionType ExperimentConditionType) *ExperimentCondition {
	condition := &ExperimentCondition{
		Type:   conditionType,
		Status: corev1.ConditionUnknown,
	}
	*condition.LastTransitionTime = metav1.Now()
	s.Conditions = append(s.Conditions, condition)
	return condition
}

func (s *ExperimentStatus) getCondition(condition ExperimentConditionType) *ExperimentCondition {
	for _, c := range s.Conditions {
		if c.Type == condition {
			return c
		}
	}

	return s.addCondition(condition)
}

// Init initialize status value of an experiment
func (e *Experiment) Init() {
	e.Status.Assessment = &Assessment{
		Baseline: VersionAssessment{
			Name:   e.Spec.Baseline,
			Weight: int32(0),
		},
		Candidates: make([]VersionAssessment, len(e.Spec.Candidates)),
	}
	for i, name := range e.Spec.Candidates {
		e.Status.Assessment.Candidates[i] = VersionAssessment{
			Name:   name,
			Weight: int32(0),
		}
	}

	// sets relevant unset conditions to Unknown state.
	for _, c := range experimentCondSet {
		e.Status.addCondition(c)
	}

	*e.Status.InitTimestamp = metav1.Now()

	if e.Status.AnalysisState.Raw == nil {
		e.Status.AnalysisState.Raw = []byte("{}")
	}

	*e.Status.Phase = PhaseProgressing
}

func (c *ExperimentCondition) markCondition(status corev1.ConditionStatus, reason, messageFormat string, messageA ...interface{}) bool {
	message := fmt.Sprintf(messageFormat, messageA...)
	updated := status != c.Status || reason != *c.Reason || message != *c.Message
	c.Status = status
	*c.Reason = reason
	*c.Message = fmt.Sprintf(messageFormat, messageA...)
	*c.LastTransitionTime = metav1.Now()
	return updated
}

// MarkMetricsSynced sets the condition that the metrics are synced with config map
// Return true if it's converted from false or unknown
func (s *ExperimentStatus) MarkMetricsSynced(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonSyncMetricsSucceeded
	return s.getCondition(ExperimentConditionMetricsSynced).
		markCondition(corev1.ConditionTrue, reason, messageFormat, messageA...), reason
}

// MarkMetricsSyncedError sets the condition that the error occurs when syncing with the config map
// Return true if it's converted from true or unknown
func (s *ExperimentStatus) MarkMetricsSyncedError(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonSyncMetricsError
	*s.Phase = PhasePause
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionMetricsSynced).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// TargetsFound returns whether status of ExperimentConditionTargetsProvided is true or not
func (s *ExperimentStatus) TargetsFound() bool {
	return s.getCondition(ExperimentConditionTargetsProvided).Status == corev1.ConditionTrue
}

// MarkTargetsFound sets the condition that the all target have been found
// Return true if it's converted from false or unknown
func (s *ExperimentStatus) MarkTargetsFound(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonTargetsFound
	return s.getCondition(ExperimentConditionTargetsProvided).
		markCondition(corev1.ConditionTrue, reason, messageFormat, messageA...), reason
}

// MarkTargetsError sets the condition that there is error in finding all targets
// Return true if it's converted from true or unknown
func (s *ExperimentStatus) MarkTargetsError(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonTargetsError
	*s.Phase = PhasePause
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionTargetsProvided).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// MarkRoutingRulesStatus sets the condition that the routing rules are ready
// Return true if it's converted from false or unknown
func (s *ExperimentStatus) MarkRoutingRulesStatus(reason, messageFormat string, messageA ...interface{}) (bool, string) {
	return s.getCondition(ExperimentConditionRoutingRulesReady).
		markCondition(corev1.ConditionTrue, reason, messageFormat, messageA...), reason
}

// MarkRoutingRulesError sets the condition that the routing rules are not ready
// Return true if it's converted from true or unknown
func (s *ExperimentStatus) MarkRoutingRulesError(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonRoutingRulesError
	*s.Phase = PhasePause
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionRoutingRulesReady).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// MarkAnalyticsServiceRunning sets the condition that the analytics service is operating normally
// Return true if it's converted from false or unknown
func (s *ExperimentStatus) MarkAnalyticsServiceRunning(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonAnalyticsServiceRunning
	return s.getCondition(ExperimentConditionAnalyticsServiceNormal).
		markCondition(corev1.ConditionTrue, reason, messageFormat, messageA...), reason
}

// MarkAnalyticsServiceError sets the condition that the analytics service breaks down
// Return true if it's converted from true or unknown
func (s *ExperimentStatus) MarkAnalyticsServiceError(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonAnalyticsServiceError
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	*s.Phase = PhasePause
	return s.getCondition(ExperimentConditionAnalyticsServiceNormal).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// MarkExperimentCompleted sets the condition that the experiemnt is completed
func (s *ExperimentStatus) MarkExperimentCompleted(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonExperimentCompleted
	*s.Phase = PhaseCompleted
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionExperimentCompleted).
		markCondition(corev1.ConditionTrue, reason, messageFormat, messageA...), reason
}

// MarkIterationUpdate sets the condition that the iteration updated
func (s *ExperimentStatus) MarkIterationUpdate(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonIterationUpdate
	*s.Phase = PhaseProgressing
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionExperimentCompleted).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// MarkTrafficUpdate sets the condition that traffic for target service updated
func (s *ExperimentStatus) MarkTrafficUpdate(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonTrafficUpdate
	*s.Phase = PhaseProgressing
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionExperimentCompleted).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// MarkExperimentPause sets the phase and status that experiment is paused by manualOverrides
// returns true if this is a newly-set operation
func (s *ExperimentStatus) MarkExperimentPause(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonActionPause
	*s.Phase = PhasePause
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionExperimentCompleted).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

// MarkExperimentResume sets the phase and status that experiment is resmued by manualOverrides
// returns true if this is a newly-set operation
func (s *ExperimentStatus) MarkExperimentResume(messageFormat string, messageA ...interface{}) (bool, string) {
	reason := ReasonActionResume
	*s.Phase = PhaseProgressing
	*s.Message = composeMessage(reason, messageFormat, messageA...)
	return s.getCondition(ExperimentConditionExperimentCompleted).
		markCondition(corev1.ConditionFalse, reason, messageFormat, messageA...), reason
}

func composeMessage(reason, messageFormat string, messageA ...interface{}) string {
	out := reason
	msg := fmt.Sprintf(messageFormat, messageA...)
	if len(msg) > 0 {
		out += ": " + msg
	}
	return out
}
