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

	corev1 "k8s.io/api/core/v1"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

func MakeRequest(object *iter8v1alpha1.Canary, baseline, canary *corev1.Service) *checkandincrement.Request {
	spec := object.Spec

	metrics := make([]checkandincrement.SuccessCriterion, len(spec.Analysis.Metrics))
	for i, metric := range spec.Analysis.Metrics {
		metrics[i] = checkandincrement.SuccessCriterion{
			MetricName: metric.Name,
			Type:       metric.Type,
			Value:      metric.Value,
			SampleSize: metric.SampleSize,
			// OnSuccess:  metric.OnSuccess,
		}
	}
	now := time.Now().Format(time.RFC3339)

	return &checkandincrement.Request{
		Baseline: checkandincrement.Window{
			StartTime: baseline.ObjectMeta.GetCreationTimestamp().Format(time.RFC3339),
			EndTime:   now,
			Tags: map[string]string{
				"destination_service_name": baseline.GetName(),
			},
		},
		Canary: checkandincrement.Window{
			StartTime: canary.ObjectMeta.GetCreationTimestamp().Format(time.RFC3339),
			EndTime:   now,
			Tags: map[string]string{
				"destination_service_name": canary.GetName(),
			},
		},
		TrafficControl: checkandincrement.TrafficControl{
			MaxTrafficPercent: 50,
			StepSize:          2,
			SuccessCriteria:   metrics,
		},
		LastState: object.Status.AnalysisState,
	}
}
