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
	Stable    = "stable"

	canaryRole = "iter8.ibm.com/role"
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

	// Remove stable rules if there is any related to the current service
	stableName := getStableName(canary)
	dr := &v1alpha3.DestinationRule{}
	if err := r.Get(context, types.NamespacedName{Name: stableName, Namespace: canary.GetNamespace()}, dr); err == nil {
		r.Delete(context, dr)
		log.Info("istio sync", "delete stable dr", stableName)
	}
	vs := &v1alpha3.VirtualService{}
	if err := r.Get(context, types.NamespacedName{Name: stableName, Namespace: canary.GetNamespace()}, vs); err == nil {
		r.Delete(context, vs)
		log.Info("istio sync", "delete stable vs", stableName)
	}
	// Set up vs and dr for experiment
	drName := getDestinationRuleName(canary)
	dr = &v1alpha3.DestinationRule{}
	if err = r.Get(context, types.NamespacedName{Name: drName, Namespace: canary.Namespace}, dr); err != nil {
		dr = newDestinationRule(canary)
		err := r.Create(context, dr)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.Info("istio-sync", "create destinationRule", drName)
	}
	vsName := getVirtualServiceName(canary)
	vs = &v1alpha3.VirtualService{}
	if err = r.Get(context, types.NamespacedName{Name: vsName, Namespace: canary.Namespace}, vs); err != nil {
		vs = makeVirtualService(0, canary)
		err := r.Create(context, vs)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.Info("istio-sync", "create vs for experiment", vsName)
	}

	baselineName, candidateName := canary.Spec.TargetService.Baseline, canary.Spec.TargetService.Candidate

	baseline, candidate := &appsv1.Deployment{}, &appsv1.Deployment{}
	// Get current deployment and candidate deployment

	if err = r.Get(context, types.NamespacedName{Name: baselineName, Namespace: serviceNamespace}, baseline); err == nil {
		log.Info("istio sync", "Found baseline in canary", baselineName)
		if updated := appendSubset(dr, baseline, Baseline); updated {
			if err := r.Update(context, dr); err != nil {
				log.Info("istio sync", "error in updating dr rule", drName)
				return reconcile.Result{}, err
			}
			log.Info("istio sync", "append subset labels to dr", dr)
		}
	}

	if err = r.Get(context, types.NamespacedName{Name: candidateName, Namespace: serviceNamespace}, candidate); err == nil {
		log.Info("istio sync", "Found candidate in canary", candidateName)
		if updated := appendSubset(dr, candidate, Candidate); updated {
			if err := r.Update(context, dr); err != nil {
				log.Info("istio sync", "error in updating dr rule", drName)
				return reconcile.Result{}, err
			}
			log.Info("istio sync", "append subset labels to dr", dr)
		}
	}

	if baseline.GetName() == "" || candidate.GetName() == "" {
		if baseline.GetName() == "" && candidate.GetName() == "" {
			log.Info("istio sync missing baseline and candidate")
			canary.Status.MarkHasNotService("Baseline and candidate deployment are missing", "")
		} else if candidate.GetName() == "" {
			log.Info("istio sync missing candidate")
			canary.Status.MarkHasNotService("Candidate deployment is missing", "")
		} else {
			log.Info("istio sync missing baseline")
			canary.Status.MarkHasNotService("Baseline deployment is missing", "")
		}
		err = r.Status().Update(context, canary)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: time.Second}, nil
	}

	log.Info("istio-sync", "baseline", baseline.GetName(), Candidate, candidate.GetName())

	// Start Canary Process
	// Get info on Canary
	traffic := canary.Spec.TrafficControl
	now := time.Now()
	// TODO: check err in getting the time value
	interval, _ := traffic.GetIntervalDuration()

	// check experiment is finished
	if canary.Spec.TrafficControl.GetIterationCount() <= canary.Status.CurrentIteration {
		log.Info("experiment completed.")
		// remove canary labels
		if err := removeCanaryLabel(context, r, baseline); err != nil {
			return reconcile.Result{}, err
		}
		if err := removeCanaryLabel(context, r, candidate); err != nil {
			return reconcile.Result{}, err
		}

		if canary.Status.AssessmentSummary.AllSuccessCriteriaMet ||
			canary.Spec.TrafficControl.Strategy == "manual" {
			// experiment is successful
			switch canary.Spec.TrafficControl.GetOnSuccess() {
			case "baseline":
				// delete routing rules
				if err := deleteRules(context, r, canary); err != nil {
					return reconcile.Result{}, err
				}
				// Set all traffic to baseline deployment
				// generate new rules to shift all traffic to baseline
				if err := setStableRules(context, r, baseline, canary); err != nil {
					return reconcile.Result{}, err
				}
			case "canary":
				// delete routing rules
				if err := deleteRules(context, r, canary); err != nil {
					return reconcile.Result{}, err
				}
				// Set all traffic to candidate deployment
				// generate new rules to shift all traffic to candidate
				if err := setStableRules(context, r, candidate, canary); err != nil {
					return reconcile.Result{}, err
				}
			case "both":
				// Change name of the current vs rule as stable
				vs.SetName(getStableName(canary))
				if err := r.Update(context, vs); err != nil {
					return reconcile.Result{}, err
				}
			}
			canary.Status.MarkRolloutCompleted()
		} else {
			// delete routing rules
			if err := deleteRules(context, r, canary); err != nil {
				return reconcile.Result{}, err
			}
			// Set all traffic to baseline deployment
			// generate new rules to shift all traffic to baseline
			if err := setStableRules(context, r, baseline, canary); err != nil {
				return reconcile.Result{}, err
			}
			canary.Status.MarkHasNotService("Experiment Failure", "")
		}

		// End experiment
		err = r.Status().Update(context, canary)
		return reconcile.Result{}, err
	}

	// Check canary rollout status
	rolloutPercent := float64(getWeight(Candidate, vs))
	log.Info("istio-sync", "prev rollout percent", rolloutPercent, "max traffic percent", traffic.GetMaxTrafficPercent())
	if now.After(canary.Status.LastIncrementTime.Add(interval)) {

		switch canary.Spec.TrafficControl.Strategy {
		case "manual":
			rolloutPercent += traffic.GetStepSize()
		case "check_and_increment":
			// Get latest analysis
			payload := MakeRequest(canary, baseline, candidate)
			response, err := checkandincrement.Invoke(log, canary.Spec.Analysis.AnalyticsService, payload)
			if err != nil {
				// TODO: Need new condition
				canary.Status.MarkHasNotService("Istio Analytics Service is not reachable", "%v", err)
				err = r.Status().Update(context, canary)
				return reconcile.Result{Requeue: true}, err
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
			canary.Status.AssessmentSummary = response.Assessment.Summary
		}

		canary.Status.CurrentIteration++
		log.Info("istio-sync", "new rollout iteration", canary.Status.CurrentIteration)
		// Increase the traffic upto max traffic amount
		if rolloutPercent <= traffic.GetMaxTrafficPercent() && getWeight(Candidate, vs) != int(rolloutPercent) {
			// Update Traffic splitting rule
			log.Info("istio-sync", "new rollout perccent", rolloutPercent)
			rv := vs.ObjectMeta.ResourceVersion
			vs = makeVirtualService(int(rolloutPercent), canary)
			setResourceVersion(rv, vs)
			log.Info("istio-sync", "updated vs", *vs)
			err := r.Update(context, vs)
			if err != nil {
				log.Info("istio-sync", "error in updating vs", vsName)
				return reconcile.Result{}, err
			}
			canary.Status.RolloutPercent = int(rolloutPercent)
		}

		canary.Status.LastIncrementTime = metav1.NewTime(now)
	}

	canary.Status.MarkRolloutNotCompleted("Progressing", "")
	err = r.Status().Update(context, canary)

	return reconcile.Result{RequeueAfter: interval}, err
}

func removeCanaryLabel(context context.Context, r *ReconcileCanary, d *appsv1.Deployment) (err error) {
	labels := d.GetLabels()
	delete(labels, canaryLabel)
	delete(labels, canaryRole)
	d.SetLabels(labels)
	if err = r.Update(context, d); err != nil {
		return
	}
	log.Info("istio sync", "remove canary label from", d.GetName())
	return
}

func deleteRules(context context.Context, r *ReconcileCanary, canary *iter8v1alpha1.Canary) (err error) {
	log := Logger(context)
	drName := getDestinationRuleName(canary)
	vsName := getVirtualServiceName(canary)

	dr := &v1alpha3.DestinationRule{}
	if err = r.Get(context, types.NamespacedName{Name: drName, Namespace: canary.Namespace}, dr); err == nil {
		if err = r.Delete(context, dr); err != nil {
			return
		}
		log.Info("istio sync", "delete dr", drName)
	} else {
		return
	}

	vs := &v1alpha3.VirtualService{}
	if err = r.Get(context, types.NamespacedName{Name: vsName, Namespace: canary.Namespace}, vs); err == nil {
		if err = r.Delete(context, vs); err != nil {
			return
		}
		log.Info("istio sync", "delete vs", vsName)
	} else {
		return
	}

	return
}

func newDestinationRule(canary *iter8v1alpha1.Canary) *v1alpha3.DestinationRule {
	dr := &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getDestinationRuleName(canary),
			Namespace: canary.Namespace,
			Labels:    map[string]string{canaryLabel: canary.ObjectMeta.Name},
		},
		Spec: v1alpha3.DestinationRuleSpec{
			Host:    canary.Spec.TargetService.Name,
			Subsets: []v1alpha3.Subset{},
		},
	}
	return dr
}

func appendSubset(dr *v1alpha3.DestinationRule, d *appsv1.Deployment, name string) bool {
	update := true
	for _, subset := range dr.Spec.Subsets {
		if subset.Name == name {
			update = false
			break
		}
	}

	if update {
		dr.Spec.Subsets = append(dr.Spec.Subsets, v1alpha3.Subset{
			Name:   name,
			Labels: d.Spec.Template.Labels,
		})
	}
	return update
}

func getDestinationRuleName(canary *iter8v1alpha1.Canary) string {
	return canary.Spec.TargetService.Name + ".iter8-canary"
}

func makeVirtualService(rolloutPercent int, canary *iter8v1alpha1.Canary) *v1alpha3.VirtualService {
	vs := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getVirtualServiceName(canary),
			Namespace: canary.Namespace,
			Labels:    map[string]string{canaryLabel: canary.ObjectMeta.Name},
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

func setStableRules(context context.Context, r *ReconcileCanary, d *appsv1.Deployment, canary *iter8v1alpha1.Canary) (err error) {
	stableDr, stableVs := newStableRules(d, canary)
	if err = r.Create(context, stableDr); err != nil {
		return
	}
	if err = r.Create(context, stableVs); err != nil {
		return
	}
	Logger(context).Info("istio-sync", "create stable dr rule", stableDr.GetName(), "stable vs rule", stableVs.GetName())
	return
}

func newStableRules(d *appsv1.Deployment, canary *iter8v1alpha1.Canary) (*v1alpha3.DestinationRule, *v1alpha3.VirtualService) {
	dr := &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStableName(canary),
			Namespace: canary.Namespace,
			Labels:    map[string]string{canaryLabel: canary.ObjectMeta.Name},
		},
		Spec: v1alpha3.DestinationRuleSpec{
			Host: canary.Spec.TargetService.Name,
			Subsets: []v1alpha3.Subset{
				v1alpha3.Subset{
					Name:   Stable,
					Labels: d.Spec.Template.Labels,
				},
			},
		},
	}

	vs := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStableName(canary),
			Namespace: canary.Namespace,
			Labels:    map[string]string{canaryLabel: canary.ObjectMeta.Name},
		},
		Spec: v1alpha3.VirtualServiceSpec{
			Hosts: []string{canary.Spec.TargetService.Name},
			HTTP: []v1alpha3.HTTPRoute{
				{
					Route: []v1alpha3.HTTPRouteDestination{
						{
							Destination: v1alpha3.Destination{
								Host:   canary.Spec.TargetService.Name,
								Subset: Stable,
							},
							Weight: 100,
						},
					},
				},
			},
		},
	}

	return dr, vs
}

func getStableName(canary *iter8v1alpha1.Canary) string {
	return canary.Spec.TargetService.Name + ".iter8-stable"
}

func (r *ReconcileCanary) finalizeIstio(context context.Context, canary *iter8v1alpha1.Canary) (reconcile.Result, error) {
	completed := canary.Status.GetCondition(iter8v1alpha1.CanaryConditionRolloutCompleted)
	if completed != nil && completed.Status != corev1.ConditionTrue {
		// Do a rollback
		// delete routing rules
		if err := deleteRules(context, r, canary); err != nil {
			return reconcile.Result{}, err
		}
		// Get baseline deployment
		baselineName := canary.Spec.TargetService.Baseline
		baseline := &appsv1.Deployment{}
		serviceNamespace := canary.Spec.TargetService.Namespace
		if serviceNamespace == "" {
			serviceNamespace = canary.Namespace
		}

		if err := r.Get(context, types.NamespacedName{Name: baselineName, Namespace: serviceNamespace}, baseline); err != nil {
			Logger(context).Info("istio sync", "Fail to get baseline deployment", baselineName)
			return reconcile.Result{}, err
		}

		// Set all traffic to baseline deployment
		// generate new rules to shift all traffic to baseline
		if err := setStableRules(context, r, baseline, canary); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, removeFinalizer(context, r, canary, Finalizer)
}
