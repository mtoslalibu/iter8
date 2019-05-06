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

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	Baseline  = "baseline"
	Candidate = "candidate"
)

func (r *ReconcileCanary) syncIstio(context context.Context, canary *iter8v1alpha1.Canary) (reconcile.Result, error) {
	serviceName := canary.Spec.TargetService.Name
	serviceNamespace := canary.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = canary.Namespace
	}

	// Get k8s service
	service := &corev1.Service{}
	err := r.Get(context, types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, service)
	if err != nil {
		canary.Status.MarkHasNotService("NotFound", "")
		err = r.Status().Update(context, canary)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	canary.Status.MarkHasService()

	// Get deployment list
	deployments := &appsv1.DeploymentList{}
	if err = r.List(context, client.MatchingLabels(service.Spec.Selector), deployments); err != nil {
		// TODO: add new type of status to set unavailable deployments
		canary.Status.MarkHasNotService("NotFound", "")
		err = r.Status().Update(context, canary)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Get current deployment and candidate deployment
	var baseline, candidate *appsv1.Deployment
	for _, d := range deployments.Items {
		if val, ok := d.ObjectMeta.Labels[canaryLabel]; ok {
			if val == Candidate {
				candidate = d.DeepCopy()
			} else if val == Baseline {
				baseline = d.DeepCopy()
			}
		}
	}

	if baseline == nil || candidate == nil {
		canary.Status.MarkHasNotService("Base or candidate deployment is missing", "")
		err = r.Status().Update(context, canary)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	log.Info("istio-sync", "baseline", baseline.GetName(), Candidate, candidate.GetName())

	// Get info on Canary
	traffic := canary.Spec.TrafficControl
	now := time.Now()
	// TODO: check err in getting the time value
	interval, _ := traffic.GetIntervalDuration()

	// Start Canary Process
	// Setup Istio Routing Rules
	// TODO: should include deployment info here
	drName := getDestinationRuleName(canary)
	dr := &v1alpha3.DestinationRule{}
	if err = r.Get(context, types.NamespacedName{Name: drName, Namespace: canary.Namespace}, dr); err != nil {
		dr = newDestinationRule(canary)
		err := r.Create(context, dr)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.Info("istio-sync", "create destinationRule", drName)
	}

	vsName := getVirtualServiceName(canary)
	vs := &v1alpha3.VirtualService{}
	if err = r.Get(context, types.NamespacedName{Name: vsName, Namespace: canary.Namespace}, vs); err != nil {
		vs = makeVirtualService(0, canary)
		//	log.Info("istio-sync", "subset name", vs.Spec.HTTP[0].Route[0].Destination.Subset)
		err := r.Create(context, vs)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.Info("istio-sync", "create virtualservice", vsName)
	}

	// Check canary rollout status
	rolloutPercent := float64(getWeight(Candidate, vs))
	log.Info("istio-sync", "prev rollout percent", rolloutPercent, "max traffic percent", traffic.GetMaxTrafficPercent())
	if rolloutPercent < traffic.GetMaxTrafficPercent() &&
		now.After(canary.Status.LastIncrementTime.Add(interval)) {

		switch canary.Spec.TrafficControl.Strategy {
		case "manual":
			rolloutPercent += traffic.GetStepSize()
		case "check_and_increment":
			// Get latest analysis
			payload := MakeRequest(canary, baseline, candidate)
			response, err := checkandincrement.Invoke(log, canary.Spec.Analysis.AnalyticsService, payload)
			if err != nil {
				// TODO: Need new condition
				canary.Status.MarkHasNotService("ErrorAnalytics", "%v", err)
				err = r.Status().Update(context, canary)
				return reconcile.Result{}, err
			}

			baselineTraffic := response.Baseline.TrafficPercentage
			canaryTraffic := response.Canary.TrafficPercentage
			log.Info("NewTraffic", "baseline", baselineTraffic, "canary", canaryTraffic)
			rolloutPercent = canaryTraffic

			lastState, err := json.Marshal(response.LastState)
			if err != nil {
				// TODO: Need new condition
				canary.Status.MarkHasNotService("ErrorAnalyticsResponse", "%v", err)
				err = r.Status().Update(context, canary)
				return reconcile.Result{}, err
			}
			canary.Status.AnalysisState = runtime.RawExtension{Raw: lastState}
		}

		log.Info("istio-sync", "new rollout perccent", rolloutPercent)
		rv := vs.ObjectMeta.ResourceVersion
		vs = makeVirtualService(int(rolloutPercent), canary)
		setResourceVersion(rv, vs)
		log.Info("istio-sync", "updated vs", *vs)
		err := r.Update(context, vs)
		if err != nil {
			return reconcile.Result{}, err
		}

		canary.Status.LastIncrementTime = metav1.NewTime(now)
	}

	result := reconcile.Result{}
	if getWeight(Candidate, vs) == int(traffic.GetMaxTrafficPercent()) {
		// Rollout done.
		canary.Status.MarkRolloutCompleted()
		canary.Status.Progressing = false
	} else {
		canary.Status.MarkRolloutNotCompleted("Progressing", "")
		canary.Status.Progressing = true
		result.RequeueAfter = interval
	}

	err = r.Status().Update(context, canary)

	return result, err
}

// apiVersion: networking.istio.io/v1alpha3
// kind: DestinationRule
// metadata:
//   name: reviews-canary
// spec:
//   host: reviews
//   subsets:
//   - name: base
// 	   labels:
// 	     iter8.ibm.com/canary: base
//   - name: candidate
// 	   labels:
// 	     iter8.ibm.com/canary: candidate

func newDestinationRule(canary *iter8v1alpha1.Canary) *v1alpha3.DestinationRule {
	dr := &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getDestinationRuleName(canary),
			Namespace: canary.Namespace,
			// TODO: add owner references
		},
		Spec: v1alpha3.DestinationRuleSpec{
			Host: canary.Spec.TargetService.Name,
			Subsets: []v1alpha3.Subset{
				v1alpha3.Subset{
					Name:   Baseline,
					Labels: map[string]string{canaryLabel: Baseline},
				},
				v1alpha3.Subset{
					Name:   Candidate,
					Labels: map[string]string{canaryLabel: Candidate},
				},
			},
		},
	}

	return dr
}

func getDestinationRuleName(canary *iter8v1alpha1.Canary) string {
	return canary.Spec.TargetService.Name + "-iter8.canary"
}

// apiVersion: networking.istio.io/v1alpha3
// kind: VirtualService
// metadata:
//   name: reviews-canary
// spec:
//   hosts:
// 	- reviews
//   http:
//   - route:
// 		- destination:
// 			host: reviews
// 			subset: base
// 	  	  weight: 50
// 		- destination:
// 			host: reviews
// 			subset: candidate
// 	  	  weight: 50

func makeVirtualService(rolloutPercent int, canary *iter8v1alpha1.Canary) *v1alpha3.VirtualService {
	vs := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getVirtualServiceName(canary),
			Namespace: canary.Namespace,
			// TODO: add owner references
		},
		Spec: v1alpha3.VirtualServiceSpec{
			Hosts: []string{canary.Spec.TargetService.Name},
			HTTP: []v1alpha3.HTTPRoute{
				{
					Route: []v1alpha3.HTTPRouteDestination{
						{
							Destination: v1alpha3.Destination{
								Host:   canary.Spec.TargetService.Name,
								Subset: Baseline,
							},
							Weight: 100 - rolloutPercent,
						},
						{
							Destination: v1alpha3.Destination{
								Host:   canary.Spec.TargetService.Name,
								Subset: Candidate,
							},
							Weight: rolloutPercent,
						},
					},
				},
			},
		},
	}

	return vs
}

// Should add deployment names
func getVirtualServiceName(canary *iter8v1alpha1.Canary) string {
	return canary.Spec.TargetService.Name + "-iter8.canary"
}

func getWeight(subset string, vs *v1alpha3.VirtualService) int {
	for _, route := range vs.Spec.HTTP[0].Route {
		if route.Destination.Subset == subset {
			return route.Weight
		}
	}
	return 0
}

func setResourceVersion(rv string, vs *v1alpha3.VirtualService) {
	vs.ObjectMeta.ResourceVersion = rv
}
