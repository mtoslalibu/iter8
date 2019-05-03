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

package canary

import (
	"context"
	"encoding/json"
	"time"

	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

func (r *ReconcileCanary) syncKnative(context context.Context, instance *iter8v1alpha1.Canary) (reconcile.Result, error) {
	// Get Knative service
	serviceName := instance.Spec.TargetService.Name
	serviceNamespace := instance.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = instance.Namespace
	}

	kservice := &servingv1alpha1.Service{}
	err := r.Get(context, types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, kservice)
	if err != nil {
		instance.Status.MarkHasNotService("NotFound", "")
		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if kservice.Spec.DeprecatedPinned != nil {
		instance.Status.MarkHasNotService("DeprecatedPinnedNotSupported", "")
		err = r.Status().Update(context, instance)
		return reconcile.Result{}, err
	}

	// link service to this canary. Only one canary can control a service
	labels := kservice.GetLabels()
	if canary, found := labels[canaryLabel]; found && canary != instance.GetName() {
		instance.Status.MarkHasNotService("ExistingCanary", "service is already controlled by %v", canary)
		err = r.Status().Update(context, instance)
		return reconcile.Result{}, err
	}

	if labels == nil {
		labels = make(map[string]string)
	}

	if _, ok := labels[canaryLabel]; !ok {
		labels[canaryLabel] = instance.GetName()
		kservice.SetLabels(labels)
		if err = r.Update(context, kservice); err != nil {
			return reconcile.Result{}, err
		}
	}

	instance.Status.MarkHasService()

	// Check mode is set to 'release'. If not, change it
	if kservice.Spec.Release == nil {
		current := getTrafficByName(kservice, "")
		if current == nil {
			instance.Status.MarkHasNotService("NoCurrentRevision", "")
			err = r.Status().Update(context, instance)
			return reconcile.Result{}, err
		}
		kservice.Spec.Release = &servingv1alpha1.ReleaseType{
			Revisions:      []string{current.RevisionName},
			RolloutPercent: 0,
			Configuration:  kservice.Spec.RunLatest.Configuration,
		}
		kservice.Spec.RunLatest = nil

		err := r.Update(context, kservice)
		if err != nil {
			instance.Status.MarkHasNotService("NotReleaseMode", "%v", err)
			err = r.Status().Update(context, instance)
			return reconcile.Result{}, err
		}
	}

	// Promote latest to candidate
	if len(kservice.Spec.Release.Revisions) == 1 {
		latest := getTrafficByName(kservice, "latest")
		if latest == nil {
			instance.Status.MarkHasNotService("NoLatestRevision", "")
			err = r.Status().Update(context, instance)
			return reconcile.Result{}, err
		}

		if kservice.Spec.Release.Revisions[0] != latest.RevisionName {
			kservice.Spec.Release.Revisions = []string{kservice.Spec.Release.Revisions[0], latest.RevisionName}
			kservice.Spec.Release.RolloutPercent = 0

			err := r.Update(context, kservice)
			if err != nil {
				instance.Status.MarkHasNotService("NoCandidate", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}
		} else {
			// latest == current. Canary is completed
			instance.Status.MarkRolloutCompleted()
			err = r.Status().Update(context, instance)
			return reconcile.Result{}, err
		}
	}

	// Check if traffic should be updated.

	traffic := instance.Spec.TrafficControl
	release := kservice.Spec.Release
	now := time.Now()
	interval, _ := traffic.GetIntervalDuration()

	if release.RolloutPercent < int(traffic.GetMaxTrafficPercent()) &&
		now.After(instance.Status.LastIncrementTime.Add(interval)) {

		newRolloutPercent := float64(release.RolloutPercent)
		switch instance.Spec.TrafficControl.Strategy {
		case "manual":
			newRolloutPercent += traffic.GetStepSize()
		case "check_and_increment":

			// Get underlying k8s services
			baselineService, err := r.getServiceForRevision(context, kservice, kservice.Spec.Release.Revisions[0])
			if err != nil {
				// TODO: maybe we want another condition
				instance.Status.MarkHasNotService("MissingCoreService", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}
			canaryService, err := r.getServiceForRevision(context, kservice, kservice.Spec.Release.Revisions[1])
			if err != nil {
				// TODO: maybe we want another condition
				instance.Status.MarkHasNotService("MissingCoreService", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}

			// Get latest analysis
			payload := MakeRequest(instance, baselineService, canaryService)
			response, err := checkandincrement.Invoke(log, instance.Spec.Analysis.AnalyticsService, payload)
			if err != nil {
				// TODO: Need new condition
				instance.Status.MarkHasNotService("ErrorAnalytics", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}

			baselineTraffic := response.Baseline.TrafficPercentage
			canaryTraffic := response.Canary.TrafficPercentage
			log.Info("NewTraffic", "baseline", baselineTraffic, "canary", canaryTraffic)
			newRolloutPercent = canaryTraffic

			lastState, err := json.Marshal(response.LastState)
			if err != nil {
				// TODO: Need new condition
				instance.Status.MarkHasNotService("ErrorAnalyticsResponse", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}
			instance.Status.AnalysisState = runtime.RawExtension{Raw: lastState}
		}

		if release.RolloutPercent != int(newRolloutPercent) {
			release.RolloutPercent = int(newRolloutPercent)

			err = r.Update(context, kservice)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		instance.Status.LastIncrementTime = metav1.NewTime(now)
	}

	result := reconcile.Result{}
	if release.RolloutPercent == int(traffic.GetMaxTrafficPercent()) {
		// Rollout done.
		instance.Status.MarkRolloutCompleted()
		instance.Status.Progressing = false
	} else {
		instance.Status.MarkRolloutNotCompleted("Progressing", "")
		instance.Status.Progressing = true
		result.RequeueAfter = interval
	}

	err = r.Status().Update(context, instance)
	return result, err
}

func getTrafficByName(service *servingv1alpha1.Service, name string) *servingv1alpha1.TrafficTarget {
	for _, traffic := range service.Status.Traffic {
		if traffic.Name == name {
			return &traffic
		}
	}
	return nil
}

func (r *ReconcileCanary) getServiceForRevision(context context.Context, ksvc *servingv1alpha1.Service, revisionName string) (*corev1.Service, error) {
	revision := &servingv1alpha1.Revision{}
	err := r.Get(context, types.NamespacedName{Name: revisionName, Namespace: ksvc.GetNamespace()}, revision)
	if err != nil {
		return nil, err
	}
	service := &corev1.Service{}
	err = r.Get(context, types.NamespacedName{Name: revision.Status.ServiceName, Namespace: ksvc.GetNamespace()}, service)
	if err != nil {
		return nil, err
	}
	return service, nil

}
