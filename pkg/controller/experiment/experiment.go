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

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/iter8-tools/iter8-controller/pkg/analytics"
	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/routing"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/targets"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func (r *ReconcileExperiment) checkExperimentCompleted(context context.Context, instance *iter8v1alpha1.Experiment) (bool, error) {
	// check experiment is finished
	if instance.Spec.TrafficControl.GetMaxIterations() < instance.Status.CurrentIteration ||
		instance.Action.TerminateExperiment() || util.ExperimentAbstract(context).Terminate() {
		if err := r.cleanUp(context, instance); err != nil {
			return true, err
		}
		return true, nil
	}

	return false, nil
}

func (r *ReconcileExperiment) cleanUp(context context.Context, instance *iter8v1alpha1.Experiment) error {
	r.iter8Cache.RemoveExperiment(instance)

	r.targets.Cleanup(context, instance, r.Client)

	if err := r.rules.Cleanup(context, instance, r.istioClient); err != nil {
		return err
	}

	r.setExperimentEndStatus(context, instance, "")
	log.Info("Cleanup of experiment is done.")
	return nil
}

func (r *ReconcileExperiment) getRoutingLists(namespace string, m map[string]string) (*v1alpha3.DestinationRuleList, *v1alpha3.VirtualServiceList, error) {
	if drl, err := r.istioClient.NetworkingV1alpha3().DestinationRules(namespace).
		List(metav1.ListOptions{LabelSelector: labels.Set(m).String()}); err != nil {

		return nil, nil, err
	} else {
		if vsl, err := r.istioClient.NetworkingV1alpha3().VirtualServices(namespace).
			List(metav1.ListOptions{LabelSelector: labels.Set(m).String()}); err == nil {
			return drl, vsl, nil

		}
		return nil, nil, err
	}
}

func (r *ReconcileExperiment) checkOrInitRules(context context.Context, instance *iter8v1alpha1.Experiment) (bool, error) {

	serviceNs := util.GetServiceNamespace(instance)
	drl, vsl, err := r.getRoutingLists(serviceNs,
		map[string]string{routing.ExperimentHost: instance.Spec.TargetService.Name})
	updateStatus := true

	if err != nil || len(drl.Items) == 0 && len(vsl.Items) == 0 {
		util.Logger(context).Info("NoRulesDetected")
		// Initialize routing rules
		if err = r.initializeRoutingRules(instance); err != nil {
			markExperimentCompleted(instance)
			r.MarkRoutingRulesError(context, instance, "Error in Initializing routing rules: %s, Experiment Ended.", err.Error())
			r.MarkExperimentFailed(context, instance, "Error in Initializing routing rules: %s, Experiment Ended.", err.Error())
			r.iter8Cache.RemoveExperiment(instance)
		} else {
			r.MarkRoutingRulesReady(context, instance,
				"Init Routing Rules Suceeded, DR: %s, VS: %s",
				r.rules.DestinationRule.GetName(), r.rules.VirtualService.GetName())
		}
	} else {
		if r.rules, err = validateDetectedRules(drl, vsl, instance); err == nil {
			r.MarkRoutingRulesReady(context, instance,
				"RoutingRules detected, DR: %s, VS: %s",
				r.rules.DestinationRule.GetName(), r.rules.VirtualService.GetName())
			updateStatus = false
		} else {
			markExperimentCompleted(instance)
			r.MarkRoutingRulesError(context, instance,
				"UnexpectedCondition: %s , Experiment Ended", err.Error())
			r.MarkExperimentFailed(context, instance, "UnexpectedCondition: %s , Experiment Ended", err.Error())
			r.iter8Cache.RemoveExperiment(instance)
		}
	}

	return updateStatus, err
}

func (r *ReconcileExperiment) initializeRoutingRules(instance *iter8v1alpha1.Experiment) (err error) {
	serviceName := instance.Spec.TargetService.Name
	serviceNamespace := util.GetServiceNamespace(instance)

	r.rules = &routing.IstioRoutingRules{}

	if instance.Spec.RoutingReference != nil {
		if err = r.detectRoutingReferences(instance); err != nil {
			return err
		}
	} else {
		// Create Dummy Stable rules
		dr := routing.NewDestinationRule(serviceName, instance.GetName(), serviceNamespace).
			WithStableLabel().
			WithInitLabel().
			Build()
		dr, err = r.istioClient.NetworkingV1alpha3().DestinationRules(serviceNamespace).Create(dr)
		if err != nil {
			return err
		}

		log.Info("NewRuleCreated", "dr", dr.GetName())

		vs := routing.NewVirtualService(serviceName, instance.GetName(), serviceNamespace).
			WithNewStableSet(serviceName).
			WithInitLabel().
			Build()
		vs, err = r.istioClient.NetworkingV1alpha3().VirtualServices(serviceNamespace).Create(vs)
		if err != nil {
			return err
		}

		log.Info("NewRuleCreated", "vs", vs.GetName())

		r.rules.DestinationRule = dr
		r.rules.VirtualService = vs

	}

	return nil
}

func (r *ReconcileExperiment) detectRoutingReferences(instance *iter8v1alpha1.Experiment) error {
	if instance.Spec.RoutingReference == nil {
		log.Info("NoExternalReference")
		return nil
	}
	// Only support single vs for edge service now
	// TODO: support DestinationRule as well
	expNamespace := instance.Namespace
	rule := instance.Spec.RoutingReference
	if rule.APIVersion == v1alpha3.SchemeGroupVersion.String() && rule.Kind == "VirtualService" {
		ruleNamespace := rule.Namespace
		if ruleNamespace == "" {
			ruleNamespace = expNamespace
		}
		vs, err := r.istioClient.NetworkingV1alpha3().VirtualServices(ruleNamespace).Get(rule.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Referenced rule does not exist: %s", err.Error())
		}

		if err := validateVirtualService(instance, vs); err != nil {
			return err
		}

		// set stable labels
		vs.SetLabels(map[string]string{
			routing.ExperimentLabel: instance.Name,
			routing.ExperimentHost:  instance.Spec.TargetService.Name,
			routing.ExperimentRole:  routing.Stable,
		})

		dr := routing.NewDestinationRule(instance.Spec.TargetService.Name, instance.GetName(), ruleNamespace).
			WithStableLabel().
			Build()

		dr, err = r.istioClient.NetworkingV1alpha3().DestinationRules(ruleNamespace).Create(dr)
		if err != nil {
			return err
		}

		r.rules.DestinationRule = dr

		// update vs
		vs, err = r.istioClient.NetworkingV1alpha3().VirtualServices(ruleNamespace).Update(vs)
		if err != nil {
			return err
		}

		r.rules.VirtualService = vs
		return nil
	}
	return fmt.Errorf("Referenced rule not supported")
}

func (r *ReconcileExperiment) setExperimentEndStatus(context context.Context, instance *iter8v1alpha1.Experiment, msg string) (err error) {
	markExperimentCompleted(instance)
	if len(msg) == 0 {
		msg = completeStatusMessage(context, instance)
	}

	if instance.Succeeded() {
		r.MarkExperimentSucceeded(context, instance, "%s", msg)
	} else {
		r.MarkExperimentFailed(context, instance, "%s", msg)
	}

	return
}

func completeStatusMessage(context context.Context, instance *iter8v1alpha1.Experiment) string {
	out := ""
	if util.ExperimentAbstract(context) != nil && util.ExperimentAbstract(context).Terminate() {
		out = util.ExperimentAbstract(context).GetTerminateStatus()
	} else if instance.Action.TerminateExperiment() {
		out = "Abort"
	} else if instance.GetStrategy() != "increment_without_check" {
		// TODO: remove this hardcoded stategy and revise the statement
		if instance.Status.AssessmentSummary.AllSuccessCriteriaMet {
			out = "All Success Criteria Were Met"
		} else {
			out = "Not All Success Criteria Were Met"
		}
	} else if instance.Spec.TrafficControl.GetMaxIterations() < instance.Status.CurrentIteration {
		out = "Last Iteration Was Completed"
	} else {
		out = "Unknown"
	}
	return out
}

func (r *ReconcileExperiment) detectTargets(context context.Context, instance *iter8v1alpha1.Experiment) (bool, error) {
	serviceName := instance.Spec.TargetService.Name
	serviceNamespace := util.GetServiceNamespace(instance)

	if r.targets == nil {
		r.targets = targets.InitTargets()
	}
	// Get k8s service
	err := r.Get(context, types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, r.targets.Service)
	if err != nil {
		r.MarkTargetsError(context, instance, "Missing Service")
		return true, fmt.Errorf("Missing service %s", serviceName)
	}

	baselineName, candidateName := instance.Spec.TargetService.Baseline, instance.Spec.TargetService.Candidate
	// Get baseline deployment and candidate deployment
	if err = r.Get(context, types.NamespacedName{Name: baselineName, Namespace: serviceNamespace}, r.targets.Baseline); err == nil {
		//	Convert state from stable to progressing
		if r.rules.IsStable() {
			// Need to pass baseline into the builder
			if err := r.rules.StableToProgressing(instance, r.targets, r.istioClient); err != nil {
				r.MarkTargetsError(context, instance, "%s", err.Error())
				return true, err
			}
		}
	} else {
		r.MarkTargetsError(context, instance, "Missing Baseline")
		return true, fmt.Errorf("Missing Baseline %s", baselineName)
	}

	if err = r.Get(context, types.NamespacedName{Name: candidateName, Namespace: serviceNamespace}, r.targets.Candidate); err == nil {
		if updateSubset(r.rules.DestinationRule, r.targets.Candidate, routing.Candidate) {
			uddr, err := r.istioClient.NetworkingV1alpha3().
				DestinationRules(r.rules.DestinationRule.GetNamespace()).
				Update(r.rules.DestinationRule)
			if err != nil {
				return false, err
			}
			r.rules.DestinationRule = uddr
		}
	} else {
		r.MarkTargetsError(context, instance, "Missing Candidate")
		return true, fmt.Errorf("Missing Candidate %s", candidateName)
	}

	if r.MarkTargetsFound(context, instance) {
		// Update start timestamp and GrafanaURL
		instance.Status.StartTimestamp = metav1.Now().UTC().UnixNano()
		updateGrafanaURL(instance, util.GetServiceNamespace(instance))
		return true, nil
	}

	return false, nil
}

// To validate whether the detected rules can be handled by the experiment
func validateDetectedRules(drl *v1alpha3.DestinationRuleList, vsl *v1alpha3.VirtualServiceList, instance *iter8v1alpha1.Experiment) (*routing.IstioRoutingRules, error) {
	out := &routing.IstioRoutingRules{}
	// should only be one set of rules for stable or progressing
	if len(drl.Items) == 1 && len(vsl.Items) == 1 {
		drrole, drok := drl.Items[0].GetLabels()[routing.ExperimentRole]
		vsrole, vsok := vsl.Items[0].GetLabels()[routing.ExperimentRole]
		if drok && vsok {
			if drrole == routing.Stable && vsrole == routing.Stable {
				// Valid stable rules detected
				out.DestinationRule = drl.Items[0].DeepCopy()
				out.VirtualService = vsl.Items[0].DeepCopy()
			} else if drrole == routing.Progressing && vsrole == routing.Progressing {
				drLabel, drok := drl.Items[0].GetLabels()[routing.ExperimentLabel]
				vsLabel, vsok := vsl.Items[0].GetLabels()[routing.ExperimentLabel]
				if drok && vsok {
					expName := instance.GetName()
					if drLabel == expName && vsLabel == expName {
						// valid progressing rules found
						out.DestinationRule = drl.Items[0].DeepCopy()
						out.VirtualService = vsl.Items[0].DeepCopy()
					} else {
						return nil, fmt.Errorf("Progressing rules of other experiment are detected")
					}
				} else {
					return nil, fmt.Errorf("Host label missing in dr or vs")
				}
			} else {
				return nil, fmt.Errorf("Invalid role specified in dr or vs")
			}
		} else {
			return nil, fmt.Errorf("experiment role label missing in dr or vs")
		}
	} else {
		return nil, fmt.Errorf("%d dr and %d vs detected", len(drl.Items), len(vsl.Items))
	}

	return out, nil
}

func validateVirtualService(instance *iter8v1alpha1.Experiment, vs *v1alpha3.VirtualService) error {
	// Look for an entry with destination host the same as target service
	if vs.Spec.Http == nil || len(vs.Spec.Http) == 0 {
		return fmt.Errorf("Empty HttpRoute")
	}

	vsNamespace, svcNamespace := vs.Namespace, instance.Spec.TargetService.Namespace
	if vsNamespace == "" {
		vsNamespace = instance.Namespace
	}
	if svcNamespace == "" {
		svcNamespace = instance.Namespace
	}

	// The first valid entry in http route is used as stable version
	for i, http := range vs.Spec.Http {
		matchIndex := -1
		for j, route := range http.Route {
			if util.EqualHost(route.Destination.Host, vsNamespace, instance.Spec.TargetService.Name, svcNamespace) {
				// Only one entry of destination is allowed in an HTTP route
				if matchIndex < 0 {
					matchIndex = j
				} else {
					return fmt.Errorf("Multiple host-matching routes found")
				}
			}
		}
		// Set 100% weight to this host
		if matchIndex >= 0 {
			vs.Spec.Http[i].Route[matchIndex].Weight = 100
			return nil
		}
	}
	return nil
}

func (r *ReconcileExperiment) progressExperiment(context context.Context, instance *iter8v1alpha1.Experiment) error {
	// Progressing on Experiment
	traffic := instance.Spec.TrafficControl
	rolloutPercent := instance.Status.TrafficSplit.Candidate
	strategy := instance.GetStrategy()
	algorithm := analytics.GetAlgorithm(strategy)
	if algorithm == nil {
		err := fmt.Errorf("Unsupported Strategy %s", strategy)
		r.MarkAnalyticsServiceError(context, instance, "%s", err.Error())
		return err
	}

	if algorithm.GetPath() == "" {
		rolloutPercent += int(traffic.GetStepSize())
	} else {
		// Get latest analysis
		payload, err := analytics.MakeRequest(instance, r.targets.Baseline, r.targets.Candidate, algorithm)
		if err != nil {
			r.MarkAnalyticsServiceError(context, instance, "%s", err.Error())
			return err
		}

		response, err := analytics.Invoke(log, instance.Spec.Analysis.GetServiceEndpoint(), payload, algorithm)
		if err != nil {
			r.MarkAnalyticsServiceError(context, instance, "%s", err.Error())
			return err
		}

		// update assessment in instance object
		instance.Status.AssessmentSummary = response.Assessment.Summary
		instance.Status.AssessmentSummary.SuccessCriteriaStatus = response.Assessment.SuccessCriteria

		if response.LastState == nil {
			instance.Status.AnalysisState.Raw = []byte("{}")
		} else {
			lastState, err := json.Marshal(response.LastState)
			if err != nil {
				r.MarkAnalyticsServiceError(context, instance, "%s", err.Error())
				return err
			}
			instance.Status.AnalysisState = runtime.RawExtension{Raw: lastState}
		}

		// abort experiment
		if response.Assessment.Summary.AbortExperiment {
			instance.Action = iter8v1alpha1.ActionOverrideFailure
			return nil
		}

		// read latest rollout percent
		rolloutPercent = int(response.Candidate.TrafficPercentage)
		r.MarkAnalyticsServiceRunning(context, instance)
	}

	// fail in this iteration
	// skip check for the first iteration
	if instance.Status.CurrentIteration > 0 && !instance.Succeeded() {
		r.MarkExperimentProgress(context, instance, true, iter8v1alpha1.ReasonIterationFailed,
			"")
	} else if rolloutPercent <= int(traffic.GetMaxTrafficPercentage()) &&
		instance.Status.TrafficSplit.Candidate != rolloutPercent {
		log.Info("NewTrafficUpdate")
		// Increase the traffic upto max traffic amount
		// Update Traffic splitting rule
		if err := r.rules.UpdateRolloutPercent(instance, rolloutPercent, r.istioClient); err != nil {
			r.MarkRoutingRulesError(context, instance, "%s", err.Error())
			return err
		}

		r.MarkExperimentProgress(context, instance, true, iter8v1alpha1.ReasonIterationSucceeded,
			"New Traffic, baseline: %d, candidate: %d",
			instance.Status.TrafficSplit.Baseline, instance.Status.TrafficSplit.Candidate)
	}

	instance.Status.LastIncrementTime = metav1.NewTime(time.Now())
	instance.Status.CurrentIteration++

	r.MarkExperimentProgress(context, instance, false, iter8v1alpha1.ReasonIterationUpdate,
		"Iteration %d Started", instance.Status.CurrentIteration)
	return nil
}

func updateSubset(dr *v1alpha3.DestinationRule, d *appsv1.Deployment, name string) bool {
	update, found := true, false
	for idx, subset := range dr.Spec.Subsets {
		if subset.Name == routing.Stable && name == routing.Baseline {
			dr.Spec.Subsets[idx].Name = name
			dr.Spec.Subsets[idx].Labels = d.Spec.Template.Labels
			found = true
			break
		}
		if subset.Name == name {
			found = true
			update = false
			break
		}
	}

	if !found {
		dr.Spec.Subsets = append(dr.Spec.Subsets, &networkingv1alpha3.Subset{
			Name:   name,
			Labels: d.Spec.Template.Labels,
		})
	}
	return update
}

func updateGrafanaURL(instance *iter8v1alpha1.Experiment, namespace string) {
	endTsStr := "now"
	if instance.Status.EndTimestamp > 0 {
		endTsStr = strconv.FormatInt(instance.Status.EndTimestamp/int64(time.Millisecond), 10)
	}
	instance.Status.GrafanaURL = instance.Spec.Analysis.GetGrafanaEndpoint() +
		"/d/eXPEaNnZz/iter8-application-metrics?" +
		"var-namespace=" + namespace +
		"&var-service=" + instance.Spec.TargetService.Name +
		"&var-baseline=" + instance.Spec.TargetService.Baseline +
		"&var-candidate=" + instance.Spec.TargetService.Candidate +
		"&from=" + strconv.FormatInt(instance.Status.StartTimestamp/int64(time.Millisecond), 10) +
		"&to=" + endTsStr
}

func markExperimentCompleted(instance *iter8v1alpha1.Experiment) {
	// Clear analysis state
	instance.Status.AnalysisState.Raw = []byte("{}")

	// Update grafana url
	instance.Status.EndTimestamp = metav1.Now().UTC().UnixNano()
	updateGrafanaURL(instance, util.GetServiceNamespace(instance))

	instance.Status.MarkExperimentCompleted()
}
