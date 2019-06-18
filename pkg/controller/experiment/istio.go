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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/knative/pkg/apis/istio/v1alpha3"
)

const (
	IstioRuleSuffix  = ".iter8-experiment"
	StableRuleSuffix = ".iter8-stable"

	Baseline    = "baseline"
	Candidate   = "candidate"
	Stable      = "stable"
	Progressing = "progressing"

	experimentRole  = "iter8.io/role"
	experimentLabel = "iter8.io/experiment"
	experimentHost  = "iter8.io/host"
)

type DestinationRuleBuilder v1alpha3.DestinationRule
type VirtualServiceBuilder v1alpha3.VirtualService

func NewDestinationRule(serviceName, name, namespace string) *DestinationRuleBuilder {
	dr := &v1alpha3.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
			Kind:       "DestinationRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + IstioRuleSuffix,
			Namespace: namespace,
			Labels: map[string]string{
				experimentLabel: name,
				experimentHost:  serviceName,
			},
		},
		Spec: v1alpha3.DestinationRuleSpec{
			Host:    serviceName,
			Subsets: []v1alpha3.Subset{},
		},
	}

	return (*DestinationRuleBuilder)(dr)
}

func (b *DestinationRuleBuilder) WithStableDeployment(d *appsv1.Deployment) *DestinationRuleBuilder {
	b.ObjectMeta.Labels[experimentRole] = Stable
	b.Spec.Subsets = append(b.Spec.Subsets, v1alpha3.Subset{
		Name:   Stable,
		Labels: d.Spec.Template.Labels,
	})

	return b
}

func (b *DestinationRuleBuilder) WithProgressingLabel() *DestinationRuleBuilder {
	b.ObjectMeta.Labels[experimentRole] = Progressing
	return b
}

func (b *DestinationRuleBuilder) Build() *v1alpha3.DestinationRule {
	return (*v1alpha3.DestinationRule)(b)
}

func NewVirtualService(serviceName, name, namespace string) *VirtualServiceBuilder {
	vs := &v1alpha3.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + IstioRuleSuffix,
			Namespace: namespace,
			Labels: map[string]string{
				experimentLabel: name,
				experimentHost:  serviceName,
			},
		},
	}

	return (*VirtualServiceBuilder)(vs)
}

func (b *VirtualServiceBuilder) WithProgressingLabel() *VirtualServiceBuilder {
	b.ObjectMeta.Labels[experimentRole] = Progressing
	return b
}

func (b *VirtualServiceBuilder) WithRolloutPercent(service string, rolloutPercent int) *VirtualServiceBuilder {
	b.Spec = v1alpha3.VirtualServiceSpec{
		Hosts: []string{service},
		HTTP: []v1alpha3.HTTPRoute{
			{
				Route: []v1alpha3.HTTPRouteDestination{
					{
						Destination: v1alpha3.Destination{
							Host:   service,
							Subset: Baseline,
						},
						Weight: 100 - rolloutPercent,
					},
					{
						Destination: v1alpha3.Destination{
							Host:   service,
							Subset: Candidate,
						},
						Weight: rolloutPercent,
					},
				},
			},
		},
	}
	return b
}

func (b *VirtualServiceBuilder) WithStableSet(service string) *VirtualServiceBuilder {
	b.ObjectMeta.Labels[experimentRole] = Stable
	b.Spec = v1alpha3.VirtualServiceSpec{
		Hosts: []string{service},
		HTTP: []v1alpha3.HTTPRoute{
			{
				Route: []v1alpha3.HTTPRouteDestination{
					{
						Destination: v1alpha3.Destination{
							Host:   service,
							Subset: Stable,
						},
						Weight: 100,
					},
				},
			},
		},
	}

	return b
}

func (b *VirtualServiceBuilder) WithResourceVersion(rv string) *VirtualServiceBuilder {
	b.ObjectMeta.ResourceVersion = rv
	return b
}

func (b *VirtualServiceBuilder) Build() *v1alpha3.VirtualService {
	return (*v1alpha3.VirtualService)(b)
}

func isStable(obj runtime.Object) (bool, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return false, err
	}

	if role, ok := accessor.GetLabels()[experimentRole]; ok {
		return (role == Stable), nil
	}
	return false, fmt.Errorf("Label %s not found", experimentRole)
}
