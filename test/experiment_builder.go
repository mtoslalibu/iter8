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

// ExperimentBuilder builds experiment object
type ExperimentBuilder struct {
	experiment *v1alpha1.Experiment
}

// NewExperiment create a new miminal experiment object
func NewExperiment(name string, namespace string) *ExperimentBuilder {
	experiment := &v1alpha1.Experiment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Experiment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: v1alpha1.ExperimentStatus{
			AnalysisState: runtime.RawExtension{Raw: []byte("{}")},
		},
	}
	experiment.Status.InitializeConditions()
	return &ExperimentBuilder{experiment: experiment}
}

// Build the experiment object
func (b *ExperimentBuilder) Build() *v1alpha1.Experiment {
	return b.experiment
}

// WithKNativeService adds KNative target service
func (b *ExperimentBuilder) WithKNativeService(name string) *ExperimentBuilder {
	b.experiment.Spec.TargetService = v1alpha1.TargetService{
		ObjectReference: &corev1.ObjectReference{
			APIVersion: "serving.knative.dev/v1alpha1",
			Name:       name,
		},
	}
	return b
}

// WithDummySuccessCriterion adds a dummy success criterion
func (b *ExperimentBuilder) WithDummySuccessCriterion() *ExperimentBuilder {
	return b.WithSuccessCriterion(v1alpha1.SuccessCriterion{
		Name:          "iter8_latency",
		ToleranceType: v1alpha1.ToleranceTypeDelta,
		Tolerance:     0.02,
	})
}

// WithSuccessCriterion adds a success criterion
func (b *ExperimentBuilder) WithSuccessCriterion(sc v1alpha1.SuccessCriterion) *ExperimentBuilder {
	b.experiment.Spec.Analysis.Metrics = append(b.experiment.Spec.Analysis.Metrics, sc)
	return b
}
