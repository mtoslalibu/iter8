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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
)

const (
	IstioRuleSuffix = ".iter8-experiment"

	Baseline    = "baseline"
	Candidate   = "candidate"
	Stable      = "stable"
	Progressing = "progressing"

	ExperimentInit  = "iter8-tools/init"
	ExperimentRole  = "iter8-tools/role"
	ExperimentLabel = "iter8-tools/experiment"
	ExperimentHost  = "iter8-tools/host"
)

type DestinationRuleBuilder v1alpha3.DestinationRule
type VirtualServiceBuilder v1alpha3.VirtualService

func NewVirtualServiceBuilder(vs *v1alpha3.VirtualService) *VirtualServiceBuilder {
	return (*VirtualServiceBuilder)(vs)
}

func NewDestinationRuleBuilder(dr *v1alpha3.DestinationRule) *DestinationRuleBuilder {
	return (*DestinationRuleBuilder)(dr)
}

func NewDestinationRule(serviceName, name, namespace string) *DestinationRuleBuilder {
	dr := &v1alpha3.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
			Kind:       "DestinationRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName + "." + namespace + IstioRuleSuffix,
			Namespace: namespace,
			Labels: map[string]string{
				ExperimentLabel: name,
				ExperimentHost:  serviceName,
			},
		},
		Spec: networkingv1alpha3.DestinationRule{
			Host:    serviceName,
			Subsets: []*networkingv1alpha3.Subset{},
		},
	}

	return (*DestinationRuleBuilder)(dr)
}

func (b *DestinationRuleBuilder) WithStableDeployment(d *appsv1.Deployment) *DestinationRuleBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Stable
	b.Spec.Subsets = append(b.Spec.Subsets, &networkingv1alpha3.Subset{
		Name:   Stable,
		Labels: d.Spec.Template.Labels,
	})

	return b
}

func (b *DestinationRuleBuilder) WithStableLabel() *DestinationRuleBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Stable
	return b
}

func (b *DestinationRuleBuilder) WithInitLabel() *DestinationRuleBuilder {
	b.ObjectMeta.Labels[ExperimentInit] = "True"
	return b
}

func (b *DestinationRuleBuilder) RemoveExperimentLabel() *DestinationRuleBuilder {
	if _, ok := b.ObjectMeta.Labels[ExperimentLabel]; ok {
		delete(b.ObjectMeta.Labels, ExperimentLabel)
	}

	if _, ok := b.ObjectMeta.Labels[ExperimentInit]; ok {
		delete(b.ObjectMeta.Labels, ExperimentInit)
	}
	return b
}

func (b *DestinationRuleBuilder) WithExperimentRegisterd(exp string) *DestinationRuleBuilder {
	b.ObjectMeta.Labels[ExperimentLabel] = exp
	return b
}

func (b *DestinationRuleBuilder) WithStableToProgressing(baseline *appsv1.Deployment) *DestinationRuleBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Progressing
	// Remove the Stable Entry
	stableIndex := -1
	for idx := range b.Spec.Subsets {
		if b.Spec.Subsets[idx].Name == Stable {
			stableIndex = idx
			break
		}
	}
	if stableIndex >= 0 {
		b.Spec.Subsets[stableIndex] = b.Spec.Subsets[0]
		b.Spec.Subsets = b.Spec.Subsets[1:]
	}

	// Add Baseline entry
	return b.WithSubset(Baseline, baseline)
}

func (b *DestinationRuleBuilder) WithProgressingToStable(stable *appsv1.Deployment) *DestinationRuleBuilder {
	b = b.WithStableLabel()
	// Remove old entries
	b.Spec.Subsets = nil
	return b.WithSubset(Stable, stable)
}

// WithSubset adds subset to the rule if not existed(will not compare subset labels)
func (b *DestinationRuleBuilder) WithSubset(name string, d *appsv1.Deployment) *DestinationRuleBuilder {
	// Omit update if subset already exists
	if b.Spec.Subsets != nil || len(b.Spec.Subsets) > 0 {
		for _, subset := range b.Spec.Subsets {
			if subset.Name == name {
				return b
			}
		}
	}

	// Add new subset to the slice
	b.Spec.Subsets = append(b.Spec.Subsets, &networkingv1alpha3.Subset{
		Name:   name,
		Labels: d.Spec.Template.Labels,
	})
	return b
}

func (b *DestinationRuleBuilder) WithProgressingLabel() *DestinationRuleBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Progressing
	return b
}

func (b *DestinationRuleBuilder) WithName(name string) *DestinationRuleBuilder {
	b.ObjectMeta.Name = name + IstioRuleSuffix
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
			Name:      serviceName + "." + namespace + IstioRuleSuffix,
			Namespace: namespace,
			Labels: map[string]string{
				ExperimentLabel: name,
				ExperimentHost:  serviceName,
			},
		},
		Spec: networkingv1alpha3.VirtualService{
			Hosts: []string{serviceName},
		},
	}

	return (*VirtualServiceBuilder)(vs)
}

func (b *VirtualServiceBuilder) WithProgressingLabel() *VirtualServiceBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Progressing
	return b
}

func (b *VirtualServiceBuilder) WithStableLabel() *VirtualServiceBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Stable
	return b
}

func (b *VirtualServiceBuilder) RemoveExperimentLabel() *VirtualServiceBuilder {
	if _, ok := b.ObjectMeta.Labels[ExperimentLabel]; ok {
		delete(b.ObjectMeta.Labels, ExperimentLabel)
	}

	if _, ok := b.ObjectMeta.Labels[ExperimentInit]; ok {
		delete(b.ObjectMeta.Labels, ExperimentInit)
	}
	return b
}

func (b *VirtualServiceBuilder) WithRolloutPercent(service, ns string, rolloutPercent int32) *VirtualServiceBuilder {
	if b.Spec.Http != nil || len(b.Spec.Http) > 0 {
		for i, http := range b.Spec.Http {
			for j, route := range http.Route {
				if util.EqualHost(route.Destination.Host, ns, service, ns) {
					if route.Destination.Subset == Baseline {
						b.Spec.Http[i].Route[j].Weight = 100 - rolloutPercent
					} else if route.Destination.Subset == Candidate {
						b.Spec.Http[i].Route[j].Weight = rolloutPercent
					}
				}
			}
		}
	} else {
		b.Spec.Hosts = []string{service}
		b.Spec.Http = append(b.Spec.Http, &networkingv1alpha3.HTTPRoute{
			Route: []*networkingv1alpha3.HTTPRouteDestination{
				{
					Destination: &networkingv1alpha3.Destination{
						Host:   service,
						Subset: Baseline,
					},
					Weight: 100 - rolloutPercent,
				},
				{
					Destination: &networkingv1alpha3.Destination{
						Host:   service,
						Subset: Candidate,
					},
					Weight: rolloutPercent,
				},
			},
		})
	}

	return b
}

func (b *VirtualServiceBuilder) AppendStableSubset(service, ns string) *VirtualServiceBuilder {
	for i, http := range b.Spec.Http {
		for j, route := range http.Route {
			if util.EqualHost(route.Destination.Host, ns, service, ns) {
				b.Spec.Http[i].Route[j].Destination.Subset = Stable
				break
			}
		}
	}
	return b
}

func (b *VirtualServiceBuilder) WithNewStableSet(service string) *VirtualServiceBuilder {
	b.ObjectMeta.Labels[ExperimentRole] = Stable
	b.Spec = networkingv1alpha3.VirtualService{
		Hosts: []string{service},
		Http: []*networkingv1alpha3.HTTPRoute{
			{
				Route: []*networkingv1alpha3.HTTPRouteDestination{
					{
						Destination: &networkingv1alpha3.Destination{
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

func (b *VirtualServiceBuilder) WithInitLabel() *VirtualServiceBuilder {
	b.ObjectMeta.Labels[ExperimentInit] = "True"
	return b
}

func (b *VirtualServiceBuilder) WithExperimentRegisterd(exp string) *VirtualServiceBuilder {
	b.ObjectMeta.Labels[ExperimentLabel] = exp
	return b
}

// WithStableToProgressing removes Stable subset while adds Baseline and Candidate subsets to the route
func (b *VirtualServiceBuilder) WithStableToProgressing(service, ns string) *VirtualServiceBuilder {
	b = b.WithProgressingLabel()
	for i, http := range b.Spec.Http {
		stableIndex := -1
		for j, route := range http.Route {
			if util.EqualHost(route.Destination.Host, ns, service, ns) {
				stableIndex = j
				break
			}
		}
		if stableIndex >= 0 {
			stablePort := b.Spec.Http[i].Route[stableIndex].Destination.Port
			// Add Baseline and Candidate entries in this HTTP section
			b.Spec.Http[i].Route = append(b.Spec.Http[i].Route, []*networkingv1alpha3.HTTPRouteDestination{
				{
					Destination: &networkingv1alpha3.Destination{
						Host:   service,
						Subset: Baseline,
						Port:   stablePort,
					},
					Weight: 100,
				},
				{
					Destination: &networkingv1alpha3.Destination{
						Host:   service,
						Subset: Candidate,
						Port:   stablePort,
					},
					Weight: 0,
				},
			}...)
			// Remove Stable entry
			b.Spec.Http[i].Route[stableIndex] = b.Spec.Http[i].Route[0]
			b.Spec.Http[i].Route = b.Spec.Http[i].Route[1:]
			break
		}
	}
	return b
}

func (b *VirtualServiceBuilder) WithProgressingToStable(service, ns string, subset string) *VirtualServiceBuilder {
	b = b.WithStableLabel()
	for i, http := range b.Spec.Http {
		stableIndex := -1
		for j, route := range http.Route {
			if util.EqualHost(route.Destination.Host, ns, service, ns) && route.Destination.Subset == subset {
				stableIndex = j
				break
			}
		}
		if stableIndex >= 0 {
			// Convert this to stable
			b.Spec.Http[i].Route[stableIndex].Destination.Subset = Stable
			b.Spec.Http[i].Route[stableIndex].Weight = 100
			// Remove other entries
			b.Spec.Http[i].Route[0] = b.Spec.Http[i].Route[stableIndex]
			b.Spec.Http[i].Route = b.Spec.Http[i].Route[:1]

			break
		}
	}
	return b
}

func (b *VirtualServiceBuilder) WithResourceVersion(rv string) *VirtualServiceBuilder {
	b.ObjectMeta.ResourceVersion = rv
	return b
}

func (b *VirtualServiceBuilder) WithName(name string) *VirtualServiceBuilder {
	b.ObjectMeta.Name = name + IstioRuleSuffix
	return b
}

func (b *VirtualServiceBuilder) Build() *v1alpha3.VirtualService {
	return (*v1alpha3.VirtualService)(b)
}

func getWeight(subset string, vs *v1alpha3.VirtualService) int32 {
	for _, route := range vs.Spec.Http[0].Route {
		if route.Destination.Subset == subset {
			return route.Weight
		}
	}
	return 0
}

func removeExperimentLabel(objs ...runtime.Object) (err error) {
	for _, obj := range objs {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return err
		}
		labels := accessor.GetLabels()
		delete(labels, ExperimentLabel)
		if _, ok := labels[ExperimentInit]; ok {
			delete(labels, ExperimentInit)
		}
		accessor.SetLabels(labels)
	}

	return nil
}
