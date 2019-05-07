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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

func MakeRequest(instance *iter8v1alpha1.Canary, baseline, canary interface{}) *checkandincrement.Request {
	spec := instance.Spec

	metrics := make([]checkandincrement.SuccessCriterion, len(spec.Analysis.Metrics))
	for i, metric := range spec.Analysis.Metrics {
		metrics[i] = checkandincrement.SuccessCriterion{
			MetricName: metric.Name,
			Type:       metric.Type,
			Value:      metric.Value,
		}
		if metric.SampleSize != nil {
			metrics[i].SampleSize = *metric.SampleSize
		}
		if metric.StopOnFailure != nil {
			metrics[i].StopOnFailure = *metric.StopOnFailure
		}
		if metric.EnableTrafficControl != nil {
			metrics[i].EnableTrafficControl = *metric.EnableTrafficControl
		}
		if metric.Confidence != nil {
			metrics[i].Confidence = *metric.Confidence
		}
	}
	now := time.Now().Format(time.RFC3339)
	destinationKey, namespaceKey, baseVal, canaryVal, baseNsVal, canaryNsVal := "", "", "", "", "", ""
	switch instance.Spec.TargetService.APIVersion {
	case KubernetesService:
		destinationKey = "destination_workload"
		namespaceKey = "destination_namespace"
		baseVal = baseline.(*appsv1.Deployment).GetName()
		canaryVal = canary.(*appsv1.Deployment).GetName()
		baseNsVal = baseline.(*appsv1.Deployment).GetNamespace()
		canaryNsVal = canary.(*appsv1.Deployment).GetNamespace()
	case KnativeServiceV1Alpha1:
		destinationKey = "destination_service_name"
		namespaceKey = "destination_service_namespace"
		baseVal = baseline.(*corev1.Service).GetName()
		canaryVal = canary.(*corev1.Service).GetName()
		baseNsVal = baseline.(*corev1.Service).GetNamespace()
		canaryNsVal = canary.(*corev1.Service).GetNamespace()
	default:
		// TODO: add err information
		return &checkandincrement.Request{}
	}

	return &checkandincrement.Request{
		Baseline: checkandincrement.Window{
			StartTime: instance.ObjectMeta.GetCreationTimestamp().Format(time.RFC3339),
			EndTime:   now,
			Tags: map[string]string{
				destinationKey: baseVal,
				namespaceKey:   baseNsVal,
			},
		},
		Canary: checkandincrement.Window{
			StartTime: instance.ObjectMeta.GetCreationTimestamp().Format(time.RFC3339),
			EndTime:   now,
			Tags: map[string]string{
				destinationKey: canaryVal,
				namespaceKey:   canaryNsVal,
			},
		},
		TrafficControl: checkandincrement.TrafficControl{
			MaxTrafficPercent: instance.Spec.TrafficControl.GetMaxTrafficPercent(),
			StepSize:          instance.Spec.TrafficControl.GetStepSize(),
			OnSuccess:         instance.Spec.TrafficControl.GetOnSuccess(),
			SuccessCriteria:   metrics,
		},
		LastState: instance.Status.AnalysisState,
	}
}
