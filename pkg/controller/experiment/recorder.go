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
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

// MarkTargetsError records the condition that the target components are missing
func (r *ReconcileExperiment) MarkTargetsError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonTargetsNotFound
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkTargetsError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkTargetsFound(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonTargetsFound
	util.Logger(context).Info(reason)
	if instance.Status.MarkTargetsFound() {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkAnalyticsServiceError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonAnalyticsServiceError
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkAnalyticsServiceError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkAnalyticsServiceRunning(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonAnalyticsServiceRunning
	util.Logger(context).Info(reason)
	if instance.Status.MarkAnalyticsServiceRunning() {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkExperimentProgress(context context.Context, instance *iter8v1alpha1.Experiment,
	broadcast bool, reason, messageFormat string, messageA ...interface{}) {

	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, messageFormat, messageA...)
	r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)

	instance.Status.MarkExperimentNotCompleted(reason, messageFormat, messageA...)
	r.markProgress()
	r.markStatusUpdate()
}

func (r *ReconcileExperiment) MarkExperimentSucceeded(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonExperimentSucceeded
	instance.Status.MarkExperimentSucceeded(reason, messageFormat, messageA...)
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, messageFormat, messageA...)
	r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	r.markStatusUpdate()
}

func (r *ReconcileExperiment) MarkExperimentFailed(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonExperimentFailed
	instance.Status.MarkExperimentFailed(reason, messageFormat, messageA...)
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
	r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
	r.markStatusUpdate()
}

func (r *ReconcileExperiment) MarkSyncMetricsError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonSyncMetricsError
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkMetricsSyncedError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkSyncMetrics(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonSyncMetricsSucceeded
	util.Logger(context).Info(reason)
	if instance.Status.MarkMetricsSynced() {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkRoutingRulesError(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonRoutingRulesError
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkRoutingRulesError(reason, messageFormat, messageA...) {
		r.eventRecorder.Eventf(instance, corev1.EventTypeWarning, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkRoutingRulesReady(context context.Context, instance *iter8v1alpha1.Experiment,
	messageFormat string, messageA ...interface{}) {
	reason := iter8v1alpha1.ReasonRoutingRulesReady
	util.Logger(context).Info(reason + ", " + fmt.Sprintf(messageFormat, messageA...))
	if instance.Status.MarkRoutingRulesReady() {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, messageFormat, messageA...)
		r.notificationCenter.Notify(instance, reason, messageFormat, messageA...)
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkActionPause(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonActionPause
	util.Logger(context).Info(reason)
	if instance.Status.MarkActionPause() {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
		r.markStatusUpdate()
	}
}

func (r *ReconcileExperiment) MarkActionResume(context context.Context, instance *iter8v1alpha1.Experiment) {
	reason := iter8v1alpha1.ReasonActionResume
	util.Logger(context).Info(reason)
	if instance.Status.MarkActionResume() {
		r.eventRecorder.Eventf(instance, corev1.EventTypeNormal, reason, "")
		r.notificationCenter.Notify(instance, reason, "")
		r.markStatusUpdate()
		// need to refresh the whole flow
		r.markRefresh()
	}
	return
}
