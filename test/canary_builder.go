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

package test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

// CanaryBuilder builds canary object
type CanaryBuilder struct {
	canary *v1alpha1.Canary
}

// NewCanary create a new miminal canary object
func NewCanary(name string, namespace string) *CanaryBuilder {
	canary := &v1alpha1.Canary{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Canary",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.CanarySpec{
			Analysis: v1alpha1.Analysis{
				Metrics: []v1alpha1.SuccessCriterion{},
			},
		},
		Status: v1alpha1.CanaryStatus{
			AnalysisState: runtime.RawExtension{Raw: []byte("{}")},
		},
	}
	canary.Status.InitializeConditions()
	return &CanaryBuilder{canary: canary}
}

// Build the canry object
func (b *CanaryBuilder) Build() *v1alpha1.Canary {
	return b.canary
}

// WithKNativeService add KNative target service
func (b *CanaryBuilder) WithKNativeService(name string) *CanaryBuilder {
	b.canary.Spec.TargetService = v1alpha1.TargetService{
		ObjectReference: &corev1.ObjectReference{
			APIVersion: "serving.knative.dev/v1alpha1",
			Name:       name,
		},
	}
	return b
}

// WithDummySuccessCriterion add a dummy success criterion
func (b *CanaryBuilder) WithDummySuccessCriterion() *CanaryBuilder {
	return b.WithSuccessCriterion(v1alpha1.SuccessCriterion{
		Name:  "iter8_latency",
		Type:  v1alpha1.SuccessCriterionDelta,
		Value: 0.02,
	})
}

// WithSuccessCriterion add a success criterion
func (b *CanaryBuilder) WithSuccessCriterion(sc v1alpha1.SuccessCriterion) *CanaryBuilder {
	b.canary.Spec.Analysis.Metrics = append(b.canary.Spec.Analysis.Metrics, sc)
	return b
}
