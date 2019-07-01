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
	"fmt"
	"strconv"
	"time"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/knative/pkg/apis/istio/v1alpha3"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileExperiment) syncKubernetes(context context.Context, instance *iter8v1alpha1.Experiment) (reconcile.Result, error) {
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
	rName := getIstioRuleName(instance)
	dr := &v1alpha3.DestinationRule{}
	vs := &v1alpha3.VirtualService{}

	// Detect stable rules with the same host
	drl := &v1alpha3.DestinationRuleList{}
	vsl := &v1alpha3.VirtualServiceList{}
	listOptions := &client.ListOptions{}
	listOptions = listOptions.
		MatchingLabels(map[string]string{experimentRole: Stable, experimentHost: serviceName}).
		InNamespace(instance.GetNamespace())
	// No need to retry if non-empty error returned(empty results are expected)
	r.List(context, listOptions, drl)
	r.List(context, listOptions, vsl)

	if len(drl.Items) == 1 && len(vsl.Items) == 1 {
		dr = drl.Items[0].DeepCopy()
		vs = vsl.Items[0].DeepCopy()
		log.Info("StableRulesFound", "dr", dr, "vs", vs)
	} else if len(drl.Items) == 0 && len(vsl.Items) == 0 {
		// Create progressing rules if not existed
		listOptions = listOptions.
			MatchingLabels(map[string]string{experimentRole: Progressing, experimentHost: serviceName}).
			InNamespace(instance.GetNamespace())
		if err := r.List(context, listOptions, drl); err != nil || len(drl.Items) == 0 {
			dr = NewDestinationRule(serviceName, instance.GetName(), instance.GetNamespace()).
				WithProgressingLabel().
				Build()
			err := r.Create(context, dr)
			if err != nil {
				return reconcile.Result{}, err
			}
			log.Info("ProgressingRuleCreated", "dr", dr)
		} else if len(drl.Items) == 1 {
			dr = drl.Items[0].DeepCopy()
			log.Info("ProgressingRuleFound", "dr", dr)
		} else {
			for _, dr := range drl.Items {
				if err := r.Delete(context, &dr); err != nil {
					return reconcile.Result{}, err
				}
			}
			return reconcile.Result{}, fmt.Errorf("UnexpectedContidtion, retrying")
		}

		if err := r.List(context, listOptions, vsl); err != nil || len(vsl.Items) == 0 {
			vs = NewVirtualService(serviceName, instance.GetName(), instance.GetNamespace()).
				WithProgressingLabel().
				WithRolloutPercent(serviceName, 0).
				Build()
			log.Info("TryingToCreateProgressingRule...", "vs", vs)
			err := r.Create(context, vs)
			if err != nil {
				return reconcile.Result{}, err
			}
			log.Info("ProgressingRuleCreated", "vs", vs)
		} else if len(vsl.Items) == 1 {
			vs = vsl.Items[0].DeepCopy()
			log.Info("ProgressingRuleFound", "vs", vs)
		} else {
			for _, vs := range vsl.Items {
				if err := r.Delete(context, &vs); err != nil {
					return reconcile.Result{}, err
				}
			}
			return reconcile.Result{}, fmt.Errorf("UnexpectedContidtion, retrying")
		}
	} else {
		//Unexpected condition, delete all before progressing rules are created
		log.Info("UnexpectedContidtion")
		if len(drl.Items) > 0 {
			for _, dr := range drl.Items {
				if err := r.Delete(context, &dr); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
		if len(vsl.Items) > 0 {
			for _, vs := range vsl.Items {
				if err := r.Delete(context, &vs); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
		return reconcile.Result{}, fmt.Errorf("UnexpectedContidtion, retrying")
	}

	stable := false
	if stable, err = isStable(dr); err != nil {
		log.Info("LabelMissingInIstioRule", err)
	}

	baselineName, candidateName := instance.Spec.TargetService.Baseline, instance.Spec.TargetService.Candidate
	baseline, candidate := &appsv1.Deployment{}, &appsv1.Deployment{}
	// Get current deployment and candidate deployment
	if err = r.Get(context, types.NamespacedName{Name: baselineName, Namespace: serviceNamespace}, baseline); err == nil {
		log.Info("BaselineDeploymentFound", "Name", baselineName)
	}

	if err = r.Get(context, types.NamespacedName{Name: candidateName, Namespace: serviceNamespace}, candidate); err == nil {
		log.Info("CandidateDeploymentFound", "Name", candidateName)
	}

	if stable && len(baseline.GetName()) > 0 {
		// Remove stable rules if there is any related to the current service
		if err := r.Delete(context, dr); err != nil {
			// Retry
			return reconcile.Result{}, err
		}
		log.Info("StableRuleDeleted", "dr", rName)

		if err := r.Delete(context, vs); err != nil {
			// Retry
			return reconcile.Result{}, err
		}
		log.Info("StableRuleDeleted", "vs", rName)

		// Create Progressing rules
		dr = NewDestinationRule(serviceName, instance.GetName(), instance.GetNamespace()).
			WithProgressingLabel().
			Build()
		if err := r.Create(context, dr); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("ProgressingRuleCreated", "dr", dr)

		vs = NewVirtualService(serviceName, instance.GetName(), instance.GetNamespace()).
			WithProgressingLabel().
			WithRolloutPercent(serviceName, 0).
			Build()
		if err = r.Create(context, vs); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("ProgressingRuleCreated", "vs", vs)
		stable = false
	}

	if !stable {
		if len(baseline.GetName()) > 0 {
			if updated := appendSubset(dr, baseline, Baseline); updated {
				if err := r.Update(context, dr); err != nil {
					log.Info("ProgressingRuleUpdateFailure", "dr", rName)
					return reconcile.Result{}, err
				}
				log.Info("ProgressingRuleUpdated", "dr", dr)
			}
		}
		if len(candidate.GetName()) > 0 {
			if updated := appendSubset(dr, candidate, Candidate); updated {
				if err := r.Update(context, dr); err != nil {
					log.Info("ProgressingRuleUpdateFailure", "dr", rName)
					return reconcile.Result{}, err
				}
				log.Info("ProgressingRuleUpdated", "dr", dr)
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

	if instance.Status.StartTimestamp == "" {
		ts := metav1.NewTime(now).UTC().UnixNano() / int64(time.Millisecond)
		instance.Status.StartTimestamp = strconv.FormatInt(ts, 10)
		updateGrafanaURL(instance, serviceNamespace)
	}

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

		ts := metav1.NewTime(now).UTC().UnixNano() / int64(time.Millisecond)
		instance.Status.EndTimestamp = strconv.FormatInt(ts, 10)
		updateGrafanaURL(instance, serviceNamespace)

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
				// Change the role of current rules as stable
				vs.ObjectMeta.SetLabels(map[string]string{experimentRole: Stable})
				dr.ObjectMeta.SetLabels(map[string]string{experimentRole: Stable})
				if err := r.Update(context, vs); err != nil {
					return reconcile.Result{}, err
				}
				if err := r.Update(context, dr); err != nil {
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
				log.Info("Istio Analytics Service is not reachable", "err", err)
				err = r.Status().Update(context, instance)
				return reconcile.Result{RequeueAfter: 5 * time.Second}, err
			}

			if response.Assessment.Summary.AbortExperiment {
				log.Info("ExperimentAborted. Rollback to Baseline.")
				// rollback to baseline and mark experiment as complelete
				// delete routing rules
				if err := deleteRules(context, r, instance); err != nil {
					return reconcile.Result{}, err
				}
				// Set all traffic to baseline deployment
				// generate new rules to shift all traffic to baseline
				if err := setStableRules(context, r, baseline, instance); err != nil {
					return reconcile.Result{}, err
				}
				instance.Status.MarkNotRollForward("AbortExperiment: Roll Back to Baseline", "")
				instance.Status.TrafficSplit.Baseline = 100
				instance.Status.TrafficSplit.Candidate = 0
				instance.Status.MarkExperimentCompleted()

				ts := metav1.NewTime(now).UTC().UnixNano() / int64(time.Millisecond)
				instance.Status.EndTimestamp = strconv.FormatInt(ts, 10)
				updateGrafanaURL(instance, serviceNamespace)
				// End experiment
				err = r.Status().Update(context, instance)
				return reconcile.Result{}, err
			}

			baselineTraffic := response.Baseline.TrafficPercentage
			candidateTraffic := response.Canary.TrafficPercentage
			log.Info("NewTraffic", "Baseline", baselineTraffic, "Candidate", candidateTraffic)
			rolloutPercent = candidateTraffic

			instance.Status.AssessmentSummary = response.Assessment.Summary
			if response.LastState == nil {
				instance.Status.AnalysisState.Raw = []byte("{}")
			} else {
				lastState, err := json.Marshal(response.LastState)
				if err != nil {
					instance.Status.MarkExperimentNotCompleted("ErrorAnalyticsResponse", "%v", err)
					err = r.Status().Update(context, instance)
					return reconcile.Result{}, err
				}
				instance.Status.AnalysisState = runtime.RawExtension{Raw: lastState}
			}
		}

		instance.Status.CurrentIteration++
		log.Info("IterationUpdated", "count", instance.Status.CurrentIteration)
		// Increase the traffic upto max traffic amount
		if rolloutPercent <= traffic.GetMaxTrafficPercentage() && getWeight(Candidate, vs) != int(rolloutPercent) {
			// Update Traffic splitting rule
			log.Info("RolloutPercentUpdated", "NewWeight", rolloutPercent)
			vs = NewVirtualService(serviceName, instance.GetName(), instance.GetNamespace()).
				WithProgressingLabel().
				WithRolloutPercent(serviceName, int(rolloutPercent)).
				WithResourceVersion(vs.ObjectMeta.ResourceVersion).
				Build()
			err := r.Update(context, vs)
			if err != nil {
				log.Info("RuleUpdateError", "vs", rName)
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

func updateGrafanaURL(instance *iter8v1alpha1.Experiment, namespace string) {
	endTs := instance.Status.EndTimestamp
	if endTs == "" {
		endTs = "now"
	}
	instance.Status.GrafanaURL = instance.Spec.Analysis.GetGrafanaEndpoint() +
		"/d/eXPEaNnZz/iter8-application-metrics?" +
		"var-namespace=" + namespace +
		"&var-service=" + instance.Spec.TargetService.Name +
		"&var-baseline=" + instance.Spec.TargetService.Baseline +
		"&var-candidate=" + instance.Spec.TargetService.Candidate +
		"&from=" + instance.Status.StartTimestamp +
		"&to=" + endTs
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
	rName := getIstioRuleName(instance)

	dr := &v1alpha3.DestinationRule{}
	if err = r.Get(context, types.NamespacedName{Name: rName, Namespace: instance.Namespace}, dr); err == nil {
		if err = r.Delete(context, dr); err != nil {
			return
		}
		log.Info("RuleDeleted", "dr", rName)
	} else {
		log.Info("RuleNotFoundWhenDeleted", "dr", rName)
	}

	vs := &v1alpha3.VirtualService{}
	if err = r.Get(context, types.NamespacedName{Name: rName, Namespace: instance.Namespace}, vs); err == nil {
		if err = r.Delete(context, vs); err != nil {
			return
		}
		log.Info("RuleDeleted", "vs", rName)
	} else {
		log.Info("RuleNotFoundWhenDeleted", "vs", rName)
	}

	return nil
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

func getWeight(subset string, vs *v1alpha3.VirtualService) int {
	for _, route := range vs.Spec.HTTP[0].Route {
		if route.Destination.Subset == subset {
			return route.Weight
		}
	}
	return 0
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
	dr := NewDestinationRule(instance.Spec.TargetService.Name, instance.GetName(), instance.GetNamespace()).
		WithStableDeployment(d).
		Build()

	vs := NewVirtualService(instance.Spec.TargetService.Name, instance.GetName(), instance.GetNamespace()).
		WithStableSet(instance.Spec.TargetService.Name).
		Build()

	return dr, vs
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

func getIstioRuleName(instance *iter8v1alpha1.Experiment) string {
	return instance.GetName() + IstioRuleSuffix
}