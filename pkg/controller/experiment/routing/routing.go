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

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/targets"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

type IstioRoutingRules struct {
	DestinationRule *v1alpha3.DestinationRule
	VirtualService  *v1alpha3.VirtualService
}

func (r *IstioRoutingRules) SetStableLabels() {
	r.DestinationRule.ObjectMeta.SetLabels(map[string]string{ExperimentRole: Stable})
	r.VirtualService.ObjectMeta.SetLabels(map[string]string{ExperimentRole: Stable})
	removeExperimentLabel(r.DestinationRule, r.VirtualService)
}

func (r *IstioRoutingRules) IsStable() bool {
	drRole, drok := r.DestinationRule.GetLabels()[ExperimentRole]
	vsRole, vsok := r.VirtualService.GetLabels()[ExperimentRole]

	return drok && vsok && drRole == Stable && vsRole == Stable
}

func (rules *IstioRoutingRules) IsInit() bool {
	_, drok := rules.DestinationRule.GetLabels()[ExperimentInit]
	_, vsok := rules.VirtualService.GetLabels()[ExperimentInit]

	return drok && vsok
}

func (r *IstioRoutingRules) StableToProgressing(instance *iter8v1alpha1.Experiment, targets *targets.Targets, ic istioclient.Interface) error {
	r.DestinationRule = NewDestinationRuleBuilder(r.DestinationRule).
		WithStableToProgressing(targets.Baseline).
		WithExperimentRegisterd(instance.Name).
		Build()

	if dr, err := ic.NetworkingV1alpha3().
		DestinationRules(r.DestinationRule.GetNamespace()).
		Update(r.DestinationRule); err != nil {
		return err
	} else {
		r.DestinationRule = dr
	}

	r.VirtualService = NewVirtualServiceBuilder(r.VirtualService).
		WithStableToProgressing(targets.Service.GetName(), util.GetServiceNamespace(instance)).
		WithExperimentRegisterd(instance.Name).
		Build()
	if vs, err := ic.NetworkingV1alpha3().
		VirtualServices(r.VirtualService.GetNamespace()).
		Update(r.VirtualService); err != nil {
		return err
	} else {
		r.VirtualService = vs
	}

	instance.Status.TrafficSplit.Baseline = 100
	instance.Status.TrafficSplit.Candidate = 0
	return nil
}

func (r *IstioRoutingRules) DeleteAll(ic istioclient.Interface) (err error) {
	if err = ic.NetworkingV1alpha3().DestinationRules(r.DestinationRule.Namespace).
		Delete(r.DestinationRule.Name, &metav1.DeleteOptions{}); err != nil {
		return
	}

	if err = ic.NetworkingV1alpha3().VirtualServices(r.VirtualService.Namespace).
		Delete(r.VirtualService.Name, &metav1.DeleteOptions{}); err != nil {
		return
	}
	return
}

func (r *IstioRoutingRules) Cleanup(context context.Context, instance *iter8v1alpha1.Experiment, ic istioclient.Interface) (err error) {
	if instance.Spec.CleanUp == iter8v1alpha1.CleanUpDelete && r.IsInit() {
		err = r.DeleteAll(ic)
	} else {
		serviceName := instance.Spec.TargetService.Name
		serviceNs := util.GetServiceNamespace(instance)

		switch util.GetStableTarget(context, instance) {
		case "baseline":
			r.ToStable(Baseline, serviceName, serviceNs)
			err = r.UpdateRemoveRules(ic)
			if err != nil {
				return
			}
			instance.Status.TrafficSplit.Baseline = 100
			instance.Status.TrafficSplit.Candidate = 0
		case "candidate":
			r.ToStable(Candidate, serviceName, serviceNs)
			err = r.UpdateRemoveRules(ic)
			if err != nil {
				return
			}
			instance.Status.TrafficSplit.Baseline = 0
			instance.Status.TrafficSplit.Candidate = 100
		case "both":
			r.SetStableLabels()
			err = r.UpdateRemoveRules(ic)
		}

	}
	return
}

func (r *IstioRoutingRules) ToStable(stableName, serviceName, serviceNamespace string) {
	r.DestinationRule = NewDestinationRuleBuilder(r.DestinationRule).
		WithProgressingToStable(stableName).
		RemoveExperimentLabel().
		Build()
	r.VirtualService = NewVirtualServiceBuilder(r.VirtualService).
		WithProgressingToStable(serviceName, serviceNamespace, stableName).
		RemoveExperimentLabel().
		Build()
}

func (r *IstioRoutingRules) UpdateRemoveRules(ic istioclient.Interface) error {
	if dr, err := ic.NetworkingV1alpha3().
		DestinationRules(r.DestinationRule.Namespace).
		Update(r.DestinationRule); err != nil {
		return err
	} else {
		r.DestinationRule = dr
	}

	if vs, err := ic.NetworkingV1alpha3().
		VirtualServices(r.VirtualService.Namespace).
		Update(r.VirtualService); err != nil {
		return err
	} else {
		r.VirtualService = vs
	}

	return nil
}

func (r *IstioRoutingRules) GetWeight(subset string) int32 {
	for _, route := range r.VirtualService.Spec.Http[0].Route {
		if route.Destination.Subset == subset {
			return route.Weight
		}
	}
	return 0
}

func (r *IstioRoutingRules) UpdateRolloutPercent(instance *iter8v1alpha1.Experiment, rolloutPercent int, ic istioclient.Interface) error {
	vs := NewVirtualServiceBuilder(r.VirtualService).
		WithRolloutPercent(instance.Spec.TargetService.Name, util.GetServiceNamespace(instance), int32(rolloutPercent)).
		Build()

	if vs, err := ic.NetworkingV1alpha3().VirtualServices(vs.Namespace).Update(vs); err != nil {
		return err
	} else {
		r.VirtualService = vs
	}

	instance.Status.TrafficSplit.Baseline = 100 - rolloutPercent
	instance.Status.TrafficSplit.Candidate = rolloutPercent
	return nil
}
