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

package routing

import (
	"context"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iter8v1alpha2 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha2"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/targets"
)

const (
	SubsetBaseline  = "iter8.baseline"
	SubsetCandidate = "iter8.candidate"
	SubsetStable    = "iter8.stable"

	RoleStable      = "stable"
	RoleProgressing = "progressing"

	ExperimentInit  = "iter8-tools/init"
	ExperimentRole  = "iter8-tools/role"
	ExperimentLabel = "iter8-tools/experiment"
	ExperimentHost  = "iter8-tools/host"

	ExternalReference = "iter8-tools/external"
)

type istioRoutingRules struct {
	destinationRule *v1alpha3.DestinationRule
	virtualService  *v1alpha3.VirtualService
}

type Router struct {
	client istioclient.Interface
	rules  istioRoutingRules
}

// Init returns an initialized router
func Init(client istioclient.Interface) *Router {
	return &Router{
		rules: istioRoutingRules{
			destinationRule: &v1alpha3.DestinationRule{},
			virtualService:  &v1alpha3.VirtualService{},
		},
		client: client,
	}
}

func (r *istioRoutingRules) isProgressing() bool {
	drRole, drok := r.DestinationRule.GetLabels()[ExperimentRole]
	vsRole, vsok := r.VirtualService.GetLabels()[ExperimentRole]

	return drok && vsok && drRole == Progressing && vsRole == Progressing
}

func (r *istioRoutingRules) isInit() bool {
	_, drok := rules.DestinationRule.GetLabels()[ExperimentInit]
	_, vsok := rules.VirtualService.GetLabels()[ExperimentInit]

	return drok && vsok
}

func (r *istioRoutingRules) isExternalReference() bool {
	_, vsok := rules.VirtualService.GetLabels()[ExternalReference]

	return vsok
}

func (r *Router) ToProgressing(instance *iter8v1alpha2.Experiment, targets *targets.Targets) (err error) {
	if r.rules.isProgressong() {
		return nil
	}

	drb := NewDestinationRuleBuilder(r.rules.destinationRule).
		InitSubsets(1+len(targets.Candidates)).
		WithSubset(targets.Baseline, SubsetBaseline, 0).
		WithProgressingLabel().
		WithExperimentRegisterd(instance.Name)

	dr, err := ic.NetworkingV1alpha3().
		DestinationRules(r.rules.destinationRule.GetNamespace()).
		Update(drb.Build())
	if err != nil {
		return
	}

	vsb := NewVirtualServiceBuilder(r.rules.virtualService).
		WithProgressingLabel().
		WithExperimentRegisterd(instance.Name)

	if r.rules.isExternalReference() {
		vsb = vsb.ExternalToProgressing(instance.Spec.Service.Name, instance.ServiceNamespace(), len(targets.Candidates))
	} else {
		vsb = vsb.ToProgressing(instance.Spec.Service.Name, len(targets.Candidates))
		trafficControl := instance.Spec.TrafficControl
		if trafficControl != nil && trafficControl.Match != nil && len(trafficControl.Match.HTTP) > 0 {
			vsb = vsb.WithHTTPMatch(trafficControl.Match.HTTP)
		}
	}

	vs, err := ic.NetworkingV1alpha3().
		VirtualServices(r.VirtualService.GetNamespace()).
		Update(vsb.Build())
	if err != nil {
		return err
	}

	r.VirtualService = vs.DeepCopy()
	r.DestinationRule = dr.DeepCopy()

	instance.Status.Assessment.Baseline.Weight = 100
	return
}

func candiateSubsetName(idx int) string {
	return SubsetCandidate + "-" + idx
}

func (r *Router) UpdateCandidates(targets *targets.Targets) (err error) {
	if r.rules.isProgressong() {
		return
	}

	drb := NewDestinationRuleBuilder(r.rules.destinationRule)
	for i, candidate := range targets.Candidates {
		drb = drb.WithSubset(candidate, candiateSubsetName(i), i+1)
	}
	dr, err := ic.NetworkingV1alpha3().
		DestinationRules(r.rules.destinationRule.GetNamespace()).
		Update(drb.Build())

	return
}

// Cleanup configures routing rules to set up traffic to desired end state
func (r *Router) Cleanup(context context.Context, instance *iter8v1alpha2.Experiment) (err error) {
	if *instance.Spec.CleanUp && r.IsInit() {
		if err = r.client.NetworkingV1alpha3().DestinationRules(r.DestinationRule.Namespace).
			Delete(r.rules.destinationRule.Name, &metav1.DeleteOptions{}); err != nil {
			return
		}

		if err = r.client.NetworkingV1alpha3().VirtualServices(r.VirtualService.Namespace).
			Delete(r.rules.virtualService.Name, &metav1.DeleteOptions{}); err != nil {
			return
		}
	} else {
		assessment := instance.Status.Assessment
		toStableSubset := make(map[string]string)
		subsetWeight := make(map[string]int32)
		switch instance.Spec.GetOnTermination() {
		case iter8v1alpha2.OnTerminationToWinner:
			if assessment != nil && assessment.Winner != nil && assessment.Winner.WinnerFound {
				// change winner version to stable
				for i, candidate := range instance.Spec.Candidates {
					if candidate == assessment.Winner.Winner {
						toStableSubset[candiateSubsetName(i)] = SubsetStable
						subsetWeight[SubsetStable] = 100
						break
					}
				}
			} else {
				// change baseline to stable
				toStableSubset[SubsetBaseline] = SubsetStable
				subsetWeight[SubsetStable] = 100
			}
		case iter8v1alpha2.OnTerminationToBaseline:
			// change baseline to stable
			toStableSubset[SubsetBaseline] = SubsetStable
			subsetWeight[SubsetStable] = 100
		case iter8v1alpha2.OnTerminationKeepLast:
			// change all subset to stable-0, stable-1...
			if assessment != nil {
				stableCnt := 0
				if assessment.Baseline.Weight > 0 {
					stableSubset := SubsetStable + "-" + stableCnt
					toStableSubset[SubsetBaseline] = stableSubset
					subsetWeight[stableSubset] = assessment.Baseline.Weight
					stableCnt++
				}
				for i, candidate := range assessment.Candidates {
					if candidate.Weight > 0 {
						stableSubset := SubsetStable + "-" + stableCnt
						toStableSubset[candiateSubsetName(i)] = stableSubset
						subsetWeight[stableSubset] = candidate.Weight
						stableCnt++
					}
				}
			}
		}

		if _, err = ic.NetworkingV1alpha3().
			DestinationRules(r.DestinationRule.Namespace).
			Update(NewDestinationRuleBuilder(r.DestinationRule).
			ProgressingToStable(toStableSubSet).
			WithStableLabel().
			RemoveExperimentLabel().
			Build()); err != nil {
			return
		}
	
		if _, err = ic.NetworkingV1alpha3().
			VirtualServices(r.VirtualService.Namespace).
			Update(NewVirtualServiceBuilder(r.VirtualService).
			ProgressingToStable(subsetWeight, instance.Spec.Service.Name, instance.ServiceNamespace()).
			WithStableLabel().
			RemoveExperimentLabel().
			Build()
		); err != nil {
			return 
		} 
	}
	return
}

// UpdateTrafficSplit updates virtualservice with latest traffic split
func (r *Router) UpdateTrafficSplit(instance *iter8v1alpha2.Experiment) error {
	subset2Weight := make(map[string]int32)
	subset2Weight[SubsetBaseline] = instance.Status.Assessment.Baseline.Weight
	for i := range instance.Spec.Candidates {
		subset2Weight[SubsetCandidate+"-"+i] = instance.Status.Assessment.Candidates[i].Weight
	}

	vs := NewVirtualServiceBuilder(r.VirtualService).
		WithTrafficSplit(instance.Spec.Service.Name, subset2Weight).
		Build()

	if vs, err := r.client.NetworkingV1alpha3().VirtualServices(vs.Namespace).Update(vs); err != nil {
		return err
	}

	r.rules.virtualService = vs.DeepCopy()

	return nil
}

// GetRoutingRules will inject routing rules into router or return error if there is any
func (r *Router) GetRoutingRules(instance *iter8v1alpha2.Experiment) error {
	selector := map[string]string{ExperimentHost: instance.Spec.Service.Name}
	drl, err := r.client.NetworkingV1alpha3().DestinationRules(instance.ServiceNamespace()).
		List(metav1.ListOptions{LabelSelector: labels.Set(selector).String()})
	if err != nil {
		return err
	}

	vsl, err := r.istioClient.NetworkingV1alpha3().VirtualServices(instance.ServiceNamespace()).
		List(metav1.ListOptions{LabelSelector: labels.Set(selector).String()})
	if err != nil {
		return err
	}

	if len(drl.Items) == 0 && len(vsl.Items) == 0 {
		// Initialize routing rules
		if err = r.router.InitRoutingRules(instance); err != nil {
			return err
		}
	} else {
		if err = r.validateDetectedRules(drl, vsl, instance); err != nil {
			return err
		}
	}
	return nil
}

// GetRoutingRuleName returns namespaced name of routing rulesin the router
func (r *Router) GetRoutingRuleName() string {
	out := ""
	if r.rules.DestinationRule != nil {
		out += "DestinationRule: " + r.rules.DestinationRule.Name + "." + r.rules.DestinationRule.Namespace
	}
	if r.rules.VirtualService != nil {
		out += ", VirtualService: " + r.rules.VirtualService.Name + "." + r.rules.VirtualService.Namespace
	}

	return out
}

// To validate whether the detected rules can be handled by the experiment
func (r *Router) validateDetectedRules(drl *v1alpha3.DestinationRuleList, vsl *v1alpha3.VirtualServiceList, instance *iter8v1alpha2.Experiment) error {
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
						r.rules.DestinationRule = drl.Items[0].DeepCopy()
						r.rules.VirtualService = vsl.Items[0].DeepCopy()
					} else {
						return fmt.Errorf("Progressing rules of other experiment are detected")
					}
				} else {
					return fmt.Errorf("Host label missing in dr or vs")
				}
			} else {
				return fmt.Errorf("Invalid role specified in dr or vs")
			}
		} else {
			return fmt.Errorf("experiment role label missing in dr or vs")
		}
	} else {
		return fmt.Errorf("%d dr and %d vs detected", len(drl.Items), len(vsl.Items))
	}

	return nil
}

// InitRoutingRules creates routing rules for experiment
func (r *Router) InitRoutingRules(instance *iter8v1alpha2.Experiment) error {
	serviceName := instance.Spec.Service.Name
	serviceNamespace := instance.ServiceNamespace()

	if instance.Spec.RoutingReference != nil {
		if err := r.detectRoutingReferences(instance); err != nil {
			return err
		}
	} else {
		dr, err := r.istioClient.NetworkingV1alpha3().DestinationRules(serviceNamespace).Create(
			NewDestinationRule(serviceName, instance.GetName(), serviceNamespace).
				WithInitLabel().
				Build())
		if err != nil {
			return err
		}

		vs, err := r.istioClient.NetworkingV1alpha3().VirtualServices(serviceNamespace).Create(
			NewVirtualService(serviceName, instance.GetName(), serviceNamespace).
				WithInitLabel().
				Build())
		if err != nil {
			return err
		}

		r.rules.DestinationRule = dr.DeepCopy()
		r.rules.VirtualService = vs.DeepCopy()
	}

	return nil
}

func (r *Router) detectRoutingReferences(instance *iter8v1alpha2.Experiment) error {
	expNamespace := instance.Namespace
	reference := instance.Spec.RoutingReference
	if reference.APIVersion == v1alpha3.SchemeGroupVersion.String() && reference.Kind == "VirtualService" {
		ruleNamespace := rule.Namespace
		if ruleNamespace == "" {
			ruleNamespace = instance.Namespace()
		}

		vs, err := r.client.NetworkingV1alpha3().VirtualServices(ruleNamespace).Get(rule.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Fail to read referenced rule: %s", err.Error())
		}

		if err := validateVirtualService(instance, vs); err != nil {
			return err
		}

		vs, err := r.istioClient.NetworkingV1alpha3().VirtualServices(ruleNamespace).Update(
			NewVirtualServiceBuilder(vs).
				WithExperimentRegistered(instance.Name).
				WithHostRegister(instance.Spec.Service.Name).
				WithExternalLabel().
				Build())
		if err != nil {
			return err
		}

		dr, err := r.istioClient.NetworkingV1alpha3().DestinationRules(ruleNamespace).Create(
			NewDestinationRule(instance.Spec.Service.Name, instance.GetName(), ruleNamespace).
				WithStableLabel().
				Build())
		if err != nil {
			return err
		}

		r.rules.DestinationRule = dr.DeepCopy()
		r.rules.VirtualService = vs.DeepCopy()
		return nil
	}
	return fmt.Errorf("Referenced rule not supported")
}

func validateVirtualService(instance *iter8v1alpha2.Experiment, vs *v1alpha3.VirtualService) error {
	// Look for an entry with destination host the same as target service
	if vs.Spec.Http == nil || len(vs.Spec.Http) == 0 {
		return fmt.Errorf("Empty HttpRoute")
	}

	vsNamespace, svcNamespace := vs.Namespace, instance.ServiceNamespace()
	if vsNamespace == "" {
		vsNamespace = instance.Namespace
	}

	// The first valid entry in http route is used as stable version
	for i, http := range vs.Spec.Http {
		matchIndex := -1
		for j, route := range http.Route {
			if util.EqualHost(route.Destination.Host, vsNamespace, instance.Spec.Service.Name, svcNamespace) {
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
