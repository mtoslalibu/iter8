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

	experimentRole = "iter8.ibm.com/role"
)

func (r *ReconcileExperiment) syncIstio(context context.Context, instance *iter8v1alpha1.Experiment) (reconcile.Result, error) {
	log := Logger(context)
	serviceName := instance.Spec.TargetService.Name
	serviceNamespace := instance.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = instance.Namespace
	}

	// Get k8s service
	service := &corev1.Service{}
	err := r.Get(context, types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, service)
	if err != nil {
		log.Info("TargetServiceNotFound", "service", serviceName)
		instance.Status.MarkHasNotService("Service Not Found", "")
		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Set up vs and dr for experiment
	drName := getDestinationRuleName(instance)
	dr := &v1alpha3.DestinationRule{}
	if err = r.Get(context, types.NamespacedName{Name: drName, Namespace: instance.Namespace}, dr); err != nil {
		dr = newDestinationRule(instance)
		err := r.Create(context, dr)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.Info("ExperimentRuleCreated", "dr", drName)
	}
	vsName := getVirtualServiceName(instance)
	vs := &v1alpha3.VirtualService{}
	if err = r.Get(context, types.NamespacedName{Name: vsName, Namespace: instance.Namespace}, vs); err != nil {
		vs = makeVirtualService(0, instance)
		err := r.Create(context, vs)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.Info("ExperimentRuleCreated", "vs", vsName)
	}

	baselineName, candidateName := instance.Spec.TargetService.Baseline, instance.Spec.TargetService.Candidate

	baseline, candidate := &appsv1.Deployment{}, &appsv1.Deployment{}
	// Get current deployment and candidate deployment

	if err = r.Get(context, types.NamespacedName{Name: baselineName, Namespace: serviceNamespace}, baseline); err == nil {
		if updated := appendSubset(dr, baseline, Baseline); updated {
			if err := r.Update(context, dr); err != nil {
				log.Info("ExperimentRuleUpdateFailure", "dr", drName)
				return reconcile.Result{}, err
			}
			log.Info("BaselineDeploymentFound", "Name", baselineName)
			log.Info("ExperimentRuleUpdated", "dr", dr)
		}
	}

	if err = r.Get(context, types.NamespacedName{Name: candidateName, Namespace: serviceNamespace}, candidate); err == nil {
		if updated := appendSubset(dr, candidate, Candidate); updated {
			if err := r.Update(context, dr); err != nil {
				log.Info("ExperimentRuleUpdateFailure", "dr", drName)
				return reconcile.Result{}, err
			}
			log.Info("CandidateDeploymentFound", "Name", candidateName)
			log.Info("ExperimentRuleUpdated", "dr", dr)
		}
	}

	stableName := getStableName(instance)
	stableDr := &v1alpha3.DestinationRule{}
	r.Get(context, types.NamespacedName{Name: stableName, Namespace: instance.GetNamespace()}, stableDr)
	stableVs := &v1alpha3.VirtualService{}
	r.Get(context, types.NamespacedName{Name: stableName, Namespace: instance.GetNamespace()}, stableVs)
	if len(stableDr.GetName()) > 0 || len(stableVs.GetName()) > 0 {
		if len(baseline.GetName()) > 0 {
			// Remove stable rules if there is any related to the current service
			if err := r.Delete(context, stableDr); err != nil {
				// Retry
				return reconcile.Result{}, err
			}
			log.Info("StableRuleDeleted", "dr", stableName)

			if err := r.Delete(context, stableVs); err != nil {
				// Retry
				return reconcile.Result{}, err
			}
			log.Info("StableRuleDeleted", "vs", stableName)
		} else {
			if err := deleteRules(context, r, instance); err != nil {
				log.Info("ExperimentRuleDeleted")
			}
		}
	}

	if baseline.GetName() == "" || candidate.GetName() == "" {
		if baseline.GetName() == "" && candidate.GetName() == "" {
			log.Info("Missing Baseline and Candidate Deployments")
			instance.Status.MarkHasNotService("Baseline and candidate deployments are missing", "")
		} else if candidate.GetName() == "" {
			log.Info("Missing Candidate Deployment")
			instance.Status.MarkHasNotService("Candidate deployment is missing", "")
		} else {
			log.Info("Missing Baseline Deployment")
			instance.Status.MarkHasNotService("Baseline deployment is missing", "")
		}

		if len(baseline.GetName()) > 0 {
			rolloutPercent := getWeight(Candidate, vs)
			instance.Status.TrafficSplit.Baseline = 100 - rolloutPercent
			instance.Status.TrafficSplit.Candidate = rolloutPercent
		}

		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	instance.Status.MarkHasService()

	// Start Experiment Process
	// Get info on Experiment
	traffic := instance.Spec.TrafficControl
	now := time.Now()
	// TODO: check err in getting the time value
	interval, _ := traffic.GetIntervalDuration()

	// check experiment is finished
	if instance.Spec.TrafficControl.GetMaxIterations() <= instance.Status.CurrentIteration {
		log.Info("ExperimentCompleted")
		// remove experiment labels
		if err := removeExperimentLabel(context, r, baseline); err != nil {
			return reconcile.Result{}, err
		}
		if err := removeExperimentLabel(context, r, candidate); err != nil {
			return reconcile.Result{}, err
		}

		// Clear analysis state
		instance.Status.AnalysisState.Raw = []byte("{}")

		if instance.Status.AssessmentSummary.AllSuccessCriteriaMet ||
			instance.Spec.TrafficControl.GetStrategy() == "increment_without_check" {
			// experiment is successful
			log.Info("ExperimentSucceeded: AllSuccessCriteriaMet")
			switch instance.Spec.TrafficControl.GetOnSuccess() {
			case "baseline":
				// delete routing rules
				if err := deleteRules(context, r, instance); err != nil {
					return reconcile.Result{}, err
				}
				// Set all traffic to baseline deployment
				// generate new rules to shift all traffic to baseline
				if err := setStableRules(context, r, baseline, instance); err != nil {
					return reconcile.Result{}, err
				}
				instance.Status.MarkNotRollForward("Roll Back to Baseline", "")
				instance.Status.TrafficSplit.Baseline = 100
				instance.Status.TrafficSplit.Candidate = 0
			case "candidate":
				// delete routing rules
				if err := deleteRules(context, r, instance); err != nil {
					return reconcile.Result{}, err
				}
				// Set all traffic to candidate deployment
				// generate new rules to shift all traffic to candidate
				if err := setStableRules(context, r, candidate, instance); err != nil {
					return reconcile.Result{}, err
				}
				instance.Status.MarkRollForward()
				instance.Status.TrafficSplit.Baseline = 0
				instance.Status.TrafficSplit.Candidate = 100
			case "both":
				// Change name of the current vs rule as stable
				vs.SetName(getStableName(instance))
				if err := r.Update(context, vs); err != nil {
					return reconcile.Result{}, err
				}
				instance.Status.MarkNotRollForward("Traffic is maintained as end of experiment", "")
			}
		} else {
			log.Info("ExperimentFailure: NotAllSuccessCriteriaMet")

			// delete routing rules
			if err := deleteRules(context, r, instance); err != nil {
				return reconcile.Result{}, err
			}
			// Set all traffic to baseline deployment
			// generate new rules to shift all traffic to baseline
			if err := setStableRules(context, r, baseline, instance); err != nil {
				return reconcile.Result{}, err
			}
			instance.Status.MarkNotRollForward("ExperimentFailure: Roll Back to Baseline", "")
			instance.Status.TrafficSplit.Baseline = 100
			instance.Status.TrafficSplit.Candidate = 0
		}

		instance.Status.MarkExperimentCompleted()
		// End experiment
		err = r.Status().Update(context, instance)
		return reconcile.Result{}, err
	}

	// Check experiment rollout status
	rolloutPercent := float64(getWeight(Candidate, vs))
	if now.After(instance.Status.LastIncrementTime.Add(interval)) {

		switch instance.Spec.TrafficControl.GetStrategy() {
		case "increment_without_check":
			rolloutPercent += traffic.GetStepSize()
		case "check_and_increment":
			// Get latest analysis
			payload := MakeRequest(instance, baseline, candidate)
			response, err := checkandincrement.Invoke(log, instance.Spec.Analysis.GetServiceEndpoint(), payload)
			if err != nil {
				instance.Status.MarkExperimentNotCompleted("Istio Analytics Service is not reachable", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{RequeueAfter: 5 * time.Second}, err
			}

			baselineTraffic := response.Baseline.TrafficPercentage
			candidateTraffic := response.Canary.TrafficPercentage
			log.Info("NewTraffic", "Baseline", baselineTraffic, "Candidate", candidateTraffic)
			rolloutPercent = candidateTraffic

			lastState, err := json.Marshal(response.LastState)
			if err != nil {
				instance.Status.MarkExperimentNotCompleted("ErrorAnalyticsResponse", "%v", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}
			instance.Status.AnalysisState = runtime.RawExtension{Raw: lastState}
			instance.Status.AssessmentSummary = response.Assessment.Summary
		}

		instance.Status.CurrentIteration++
		log.Info("IterationUpdated", "count", instance.Status.CurrentIteration)
		// Increase the traffic upto max traffic amount
		if rolloutPercent <= traffic.GetMaxTrafficPercentage() && getWeight(Candidate, vs) != int(rolloutPercent) {
			// Update Traffic splitting rule
			log.Info("RolloutPercentUpdated", "NewWeight", rolloutPercent)
			rv := vs.ObjectMeta.ResourceVersion
			vs = makeVirtualService(int(rolloutPercent), instance)
			setResourceVersion(rv, vs)
			err := r.Update(context, vs)
			if err != nil {
				log.Info("RuleUpdateError", "vs", vsName)
				return reconcile.Result{}, err
			}
			instance.Status.TrafficSplit.Baseline = 100 - int(rolloutPercent)
			instance.Status.TrafficSplit.Candidate = int(rolloutPercent)
		}
	}

	instance.Status.LastIncrementTime = metav1.NewTime(now)

	instance.Status.MarkExperimentNotCompleted("Progressing", "")
	err = r.Status().Update(context, instance)
	return reconcile.Result{RequeueAfter: interval}, err
}

func removeExperimentLabel(context context.Context, r *ReconcileExperiment, d *appsv1.Deployment) (err error) {
	labels := d.GetLabels()
	delete(labels, experimentLabel)
	delete(labels, experimentRole)
	d.SetLabels(labels)
	if err = r.Update(context, d); err != nil {
		return
	}
	Logger(context).Info("ExperimentLabelRemoved", "deployment", d.GetName())
	return
}

func deleteRules(context context.Context, r *ReconcileExperiment, instance *iter8v1alpha1.Experiment) (err error) {
	log := Logger(context)
	drName := getDestinationRuleName(instance)
	vsName := getVirtualServiceName(instance)

	dr := &v1alpha3.DestinationRule{}
	if err = r.Get(context, types.NamespacedName{Name: drName, Namespace: instance.Namespace}, dr); err == nil {
		if err = r.Delete(context, dr); err != nil {
			return
		}
		log.Info("RuleDeleted", "dr", drName)
	} else {
		log.Info("RuleNotFoundWhenDeleted", "dr", drName)
	}

	vs := &v1alpha3.VirtualService{}
	if err = r.Get(context, types.NamespacedName{Name: vsName, Namespace: instance.Namespace}, vs); err == nil {
		if err = r.Delete(context, vs); err != nil {
			return
		}
		log.Info("RuleDeleted", "vs", drName)
	} else {
		log.Info("RuleNotFoundWhenDeleted", "vs", vsName)
	}

	return nil
}

func newDestinationRule(instance *iter8v1alpha1.Experiment) *v1alpha3.DestinationRule {
	dr := &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getDestinationRuleName(instance),
			Namespace: instance.Namespace,
			Labels:    map[string]string{experimentLabel: instance.ObjectMeta.Name},
		},
		Spec: v1alpha3.DestinationRuleSpec{
			Host:    instance.Spec.TargetService.Name,
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

func getDestinationRuleName(instance *iter8v1alpha1.Experiment) string {
	return instance.Spec.TargetService.Name + ".iter8-experiment"
}

func makeVirtualService(rolloutPercent int, instance *iter8v1alpha1.Experiment) *v1alpha3.VirtualService {
	vs := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getVirtualServiceName(instance),
			Namespace: instance.Namespace,
			Labels:    map[string]string{experimentLabel: instance.ObjectMeta.Name},
		},
		Spec: v1alpha3.VirtualServiceSpec{
			Hosts: []string{instance.Spec.TargetService.Name},
			HTTP: []v1alpha3.HTTPRoute{
				{
					Route: []v1alpha3.HTTPRouteDestination{
						{
							Destination: v1alpha3.Destination{
								Host:   instance.Spec.TargetService.Name,
								Subset: Baseline,
							},
							Weight: 100 - rolloutPercent,
						},
						{
							Destination: v1alpha3.Destination{
								Host:   instance.Spec.TargetService.Name,
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

func getVirtualServiceName(instance *iter8v1alpha1.Experiment) string {
	return instance.Spec.TargetService.Name + ".iter8-experiment"
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

func setStableRules(context context.Context, r *ReconcileExperiment, d *appsv1.Deployment, instance *iter8v1alpha1.Experiment) (err error) {
	stableDr, stableVs := newStableRules(d, instance)
	if err = r.Create(context, stableDr); err != nil {
		return
	}
	if err = r.Create(context, stableVs); err != nil {
		return
	}
	Logger(context).Info("StableRulesCreated", " dr", stableDr.GetName(), "vs", stableVs.GetName())
	return
}

func newStableRules(d *appsv1.Deployment, instance *iter8v1alpha1.Experiment) (*v1alpha3.DestinationRule, *v1alpha3.VirtualService) {
	dr := &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStableName(instance),
			Namespace: instance.Namespace,
			Labels:    map[string]string{experimentLabel: instance.ObjectMeta.Name},
		},
		Spec: v1alpha3.DestinationRuleSpec{
			Host: instance.Spec.TargetService.Name,
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
			Name:      getStableName(instance),
			Namespace: instance.Namespace,
			Labels:    map[string]string{experimentLabel: instance.ObjectMeta.Name},
		},
		Spec: v1alpha3.VirtualServiceSpec{
			Hosts: []string{instance.Spec.TargetService.Name},
			HTTP: []v1alpha3.HTTPRoute{
				{
					Route: []v1alpha3.HTTPRouteDestination{
						{
							Destination: v1alpha3.Destination{
								Host:   instance.Spec.TargetService.Name,
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

func getStableName(instance *iter8v1alpha1.Experiment) string {
	return instance.Spec.TargetService.Name + ".iter8-stable"
}

func (r *ReconcileExperiment) finalizeIstio(context context.Context, instance *iter8v1alpha1.Experiment) (reconcile.Result, error) {
	completed := instance.Status.GetCondition(iter8v1alpha1.ExperimentConditionExperimentCompleted)
	if completed != nil && completed.Status != corev1.ConditionTrue {
		// Do a rollback
		// delete routing rules
		if err := deleteRules(context, r, instance); err != nil {
			return reconcile.Result{}, err
		}
		// Get baseline deployment
		baselineName := instance.Spec.TargetService.Baseline
		baseline := &appsv1.Deployment{}
		serviceNamespace := instance.Spec.TargetService.Namespace
		if serviceNamespace == "" {
			serviceNamespace = instance.Namespace
		}

		if err := r.Get(context, types.NamespacedName{Name: baselineName, Namespace: serviceNamespace}, baseline); err != nil {
			Logger(context).Info("BaselineNotFoundWhenDeleted", "name", baselineName)
		} else {
			// Set all traffic to baseline deployment
			// generate new rules to shift all traffic to baseline
			if err := setStableRules(context, r, baseline, instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, removeFinalizer(context, r, instance, Finalizer)
}
