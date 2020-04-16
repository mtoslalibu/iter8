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

package experiment

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
)

// MarkTargetsError records the condition that the target components are missing
func (r *ReconcileExperiment) MarkTargetsError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonTargetsNotFound
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkTargetsError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	}
}

func (r *ReconcileExperiment) MarkTargetsFound(context context.Context, instance *iter8v1alpha1.Experiment) bool {
	reason := iter8v1alpha1.ReasonTargetsFound
	value := instance.Status.MarkTargetsFound()
	if value {
		r.recordNormalEvent(true, instance, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
	}
	Logger(context).Info("All targets are found.")
	return value
}

func (r *ReconcileExperiment) MarkAnalyticsServiceError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonAnalyticsServiceError
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkAnalyticsServiceError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	}
}

func (r *ReconcileExperiment) MarkAnalyticsServiceRunning(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonAnalyticsServiceRunning
	Logger(context).Info(reason)
	if instance.Status.MarkAnalyticsServiceRunning() {
		r.recordNormalEvent(true, instance, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
	}
}

func (r *ReconcileExperiment) MarkExperimentProgress(context context.Context, instance *iter8v1alpha1.Experiment,
	broadcast bool, reason, messageFormat string, messageA ...interface{}) {
	instance.Status.MarkExperimentNotCompleted(reason, messageFormat, messageA...)
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	r.recordNormalEvent(broadcast, instance, reason, messageFormat, messageA...)
	r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
}

func (r *ReconcileExperiment) MarkExperimentSucceeded(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonExperimentSucceeded
	instance.Status.MarkExperimentSucceeded(reason, messageFormat, messageA...)
	markExperimentCompleted(instance)
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	r.recordNormalEvent(true, instance, reason, messageFormat, messageA...)
	r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
}

func (r *ReconcileExperiment) MarkExperimentFailed(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonExperimentFailed
	instance.Status.MarkExperimentFailed(reason, messageFormat, messageA...)
	markExperimentCompleted(instance)
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
	r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
}

func (r *ReconcileExperiment) MarkSyncMetricsError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonSyncMetricsError
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkMetricsSyncedError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	}
}

func (r *ReconcileExperiment) MarkSyncMetrics(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonSyncMetricsSucceeded
	Logger(context).Info(reason)
	if instance.Status.MarkMetricsSynced() {
		r.recordNormalEvent(true, instance, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
	}
}

func (r *ReconcileExperiment) MarkRoutingRulesError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonRoutingRulesError
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkRoutingRulesError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	}
}

func (r *ReconcileExperiment) MarkRoutingRulesReady(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonRoutingRulesReady
	Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkRoutingRulesReady() {
		r.recordNormalEvent(true, instance, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	}
}

func (r *ReconcileExperiment) recordNormalEvent(broadcast bool, instance *iter8v1alpha1.Experiment, reason string,
	messageFormat string, messageA ...interface{}) {
	if broadcast {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, messageFormat, messageA...)
	}
}
