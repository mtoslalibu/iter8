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
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iter8v1alpha2 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha2"
)

const (
	IstioRuleSuffix = ".iter8-experiment"
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

func (b *DestinationRuleBuilder) WithStableLabel() *DestinationRuleBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentRole] = RoleStable
	return b
}

func (b *DestinationRuleBuilder) WithProgressingLabel() *DestinationRuleBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentRole] = RoleProgressing
	return b
}

func (b *DestinationRuleBuilder) WithInitLabel() *DestinationRuleBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
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

func (b *DestinationRuleBuilder) WithExperimentRegistered(exp string) *DestinationRuleBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentLabel] = exp
	return b
}

func (b *DestinationRuleBuilder) WithHostRegistered(host string) *DestinationRuleBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentHost] = host
	return b
}

func (b *DestinationRuleBuilder) InitSubsets(count int) *DestinationRuleBuilder {
	b.Spec.Subsets = make([]*networkingv1alpha3.Subset, count)
	return b
}

// WithSubset converts stable dr to progressing dr
func (b *DestinationRuleBuilder) WithSubset(d *appsv1.Deployment, subsetName string, idx int) *DestinationRuleBuilder {
	for idx >= len(b.Spec.Subsets) {
		b.Spec.Subsets = append(b.Spec.Subsets, &networkingv1alpha3.Subset{})
	}
	b.Spec.Subsets[idx] = &networkingv1alpha3.Subset{
		Name:   subsetName,
		Labels: d.Spec.Template.ObjectMeta.Labels,
	}

	return b
}

func (b *DestinationRuleBuilder) ProgressingToStable(stableSubsets map[string]string) *DestinationRuleBuilder {
	cnt := 0
	for i := 0; i < len(b.Spec.Subsets); i++ {
		if stableName, ok := stableSubsets[b.Spec.Subsets[i].Name]; ok {
			subset := b.Spec.Subsets[i]
			subset.Name = stableName
			b.Spec.Subsets[cnt] = subset
			cnt++
		}
	}

	b.Spec.Subsets = b.Spec.Subsets[:cnt]
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

func (b *VirtualServiceBuilder) WithInitLabel() *VirtualServiceBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentInit] = "True"
	return b
}

func (b *VirtualServiceBuilder) WithProgressingLabel() *VirtualServiceBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentRole] = RoleProgressing
	return b
}

func (b *VirtualServiceBuilder) WithStableLabel() *VirtualServiceBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentRole] = RoleStable
	return b
}

func (b *VirtualServiceBuilder) WithExperimentRegistered(exp string) *VirtualServiceBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentLabel] = exp
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

// WithHTTPRoute updates the first http route
func (b *VirtualServiceBuilder) WithHTTPRoute(route *networkingv1alpha3.HTTPRoute) *VirtualServiceBuilder {
	if b.Spec.Http == nil || len(b.Spec.Http) == 0 {
		b.Spec.Http = append(b.Spec.Http, &networkingv1alpha3.HTTPRoute{
			Name: "iter8-route",
		})
	}

	b.Spec.Http[0] = route

	return b
}

// WithTrafficSplit will update http route with specified traffic split
func (b *VirtualServiceBuilder) WithTrafficSplit(host string, trafficSplit map[string]int32) *VirtualServiceBuilder {
	b.Spec.Http = make([]*networkingv1alpha3.HTTPRoute, 0)
	for name, traffic := range trafficSplit {
		b.Spec.Http = append(b.Spec.Http, &networkingv1alpha3.HTTPRoute{
			Route: []*networkingv1alpha3.HTTPRouteDestination{
				{
					Destination: &networkingv1alpha3.Destination{
						Host:   host,
						Subset: name,
					},
					Weight: traffic,
				},
			},
		})
	}

	return b
}

func (b *VirtualServiceBuilder) WithHTTPMatch(httpMatch []*iter8v1alpha2.HTTPMatchRequest) *VirtualServiceBuilder {
	if b.Spec.Http == nil || len(b.Spec.Http) == 0 {
		b.Spec.Http = append(b.Spec.Http, &networkingv1alpha3.HTTPRoute{})
	}
	for _, match := range httpMatch {
		b.Spec.Http[0].Match = append(b.Spec.Http[0].Match, convertMatchToIstio(match))
	}
	return b
}

func (b *VirtualServiceBuilder) ToProgressing(service string, candidateCount int) *VirtualServiceBuilder {
	if b.Spec.Http == nil || len(b.Spec.Http) == 0 {
		b.Spec.Http = append(b.Spec.Http, &networkingv1alpha3.HTTPRoute{})
	}

	b.Spec.Http[0].Route = make([]*networkingv1alpha3.HTTPRouteDestination, candidateCount+1)
	b.Spec.Http[0].Route[0] = &networkingv1alpha3.HTTPRouteDestination{
		Destination: &networkingv1alpha3.Destination{
			Host:   service,
			Subset: SubsetBaseline,
		},
		Weight: 100,
	}

	for i := 0; i < candidateCount; i++ {
		b.Spec.Http[0].Route[i+1] = &networkingv1alpha3.HTTPRouteDestination{
			Destination: &networkingv1alpha3.Destination{
				Host:   service,
				Subset: candiateSubsetName(i),
			},
			Weight: 0,
		}
	}

	return b
}

func (b *VirtualServiceBuilder) ProgressingToStable(weight map[string]int32, host, ns string) *VirtualServiceBuilder {
	if len(b.Spec.Http) == 0 {
		b.Spec.Http = append(b.Spec.Http, NewEmptyHTTPRoute().Build())
	}
	http := b.Spec.Http[0]
	http.Route = make([]*networkingv1alpha3.HTTPRouteDestination, len(weight))

	i := 0
	for name, w := range weight {
		http.Route[i] = &networkingv1alpha3.HTTPRouteDestination{
			Destination: &networkingv1alpha3.Destination{
				Host:   host,
				Subset: name,
			},
			Weight: w,
		}
		i++
	}
	return b
}

func (b *VirtualServiceBuilder) WithHostRegistered(host string) *VirtualServiceBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExperimentHost] = host
	return b
}

func (b *VirtualServiceBuilder) WithExternalLabel() *VirtualServiceBuilder {
	if b.ObjectMeta.GetLabels() == nil {
		b.ObjectMeta.SetLabels(map[string]string{})
	}
	b.ObjectMeta.Labels[ExternalReference] = "True"
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

func convertMatchToIstio(m *iter8v1alpha2.HTTPMatchRequest) *networkingv1alpha3.HTTPMatchRequest {
	out := &networkingv1alpha3.HTTPMatchRequest{
		Name:          m.Name,
		Port:          m.Port,
		IgnoreUriCase: m.IgnoreURICase,
	}

	if m.URI != nil && m.URI.IsValid() {
		out.Uri = toStringMatchExact(m.URI)
	}

	return out
}

func toStringMatchExact(s *iter8v1alpha2.StringMatch) *networkingv1alpha3.StringMatch {
	if s.Exact != nil {
		return &networkingv1alpha3.StringMatch{
			MatchType: &networkingv1alpha3.StringMatch_Exact{
				Exact: *(s.Exact)},
		}
	}
	if s.Prefix != nil {
		return &networkingv1alpha3.StringMatch{
			MatchType: &networkingv1alpha3.StringMatch_Prefix{
				Prefix: *(s.Prefix)},
		}
	}

	return &networkingv1alpha3.StringMatch{
		MatchType: &networkingv1alpha3.StringMatch_Regex{
			Regex: *(s.Regex)},
	}
}

type HTTPRouteBuilder networkingv1alpha3.HTTPRoute

func NewEmptyHTTPRoute() *HTTPRouteBuilder {
	return (*HTTPRouteBuilder)(&networkingv1alpha3.HTTPRoute{})
}

func NewHTTPRoute(route *networkingv1alpha3.HTTPRoute) *HTTPRouteBuilder {
	return (*HTTPRouteBuilder)(route)
}

func (b *HTTPRouteBuilder) WithDestination(d *networkingv1alpha3.HTTPRouteDestination) *HTTPRouteBuilder {
	b.Route = append(b.Route, d)
	return b
}

func (b *HTTPRouteBuilder) ClearRoute() *HTTPRouteBuilder {
	b.Route = make([]*networkingv1alpha3.HTTPRouteDestination, 0)
	return b
}

func (b *HTTPRouteBuilder) Build() *networkingv1alpha3.HTTPRoute {
	return (*networkingv1alpha3.HTTPRoute)(b)
}

type HTTPRouteDestinationBuilder networkingv1alpha3.HTTPRouteDestination

func NewHTTPRouteDestination() *HTTPRouteDestinationBuilder {
	return (*HTTPRouteDestinationBuilder)(&networkingv1alpha3.HTTPRouteDestination{
		Destination: &networkingv1alpha3.Destination{},
	})
}

func (b *HTTPRouteDestinationBuilder) WithWeight(w int32) *HTTPRouteDestinationBuilder {
	b.Weight = w
	return b
}

func (b *HTTPRouteDestinationBuilder) WithHost(host string) *HTTPRouteDestinationBuilder {
	b.Destination.Host = host
	return b
}

func (b *HTTPRouteDestinationBuilder) WithSubset(subset string) *HTTPRouteDestinationBuilder {
	b.Destination.Subset = subset
	return b
}

func (b *HTTPRouteDestinationBuilder) WithPort(port uint32) *HTTPRouteDestinationBuilder {
	b.Destination.Port = &networkingv1alpha3.PortSelector{
		Number: port,
	}
	return b
}

func (b *HTTPRouteDestinationBuilder) Build() *networkingv1alpha3.HTTPRouteDestination {
	return (*networkingv1alpha3.HTTPRouteDestination)(b)
}
