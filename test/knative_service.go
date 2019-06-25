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
	"fmt"

	servingalpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	servingbeta1 "github.com/knative/serving/pkg/apis/serving/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// KnativeServiceBuilder builds Knative service object
type KnativeServiceBuilder servingalpha1.Service

// NewKnativeService creates a default Knative service with one revision
func NewKnativeService(name string, namespace string) *KnativeServiceBuilder {
	s := &servingalpha1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: servingalpha1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servingalpha1.ServiceSpec{
			ConfigurationSpec: servingalpha1.ConfigurationSpec{
				Template: &servingalpha1.RevisionTemplateSpec{},
			},
			RouteSpec: servingalpha1.RouteSpec{
				Traffic: []servingalpha1.TrafficTarget{},
			},
		},
	}
	//	s.SetDefaults(context.TODO())
	return (*KnativeServiceBuilder)(s)
}

// Build the Knative Service object
func (b *KnativeServiceBuilder) Build() *servingalpha1.Service {
	return (*servingalpha1.Service)(b)
}

func (b *KnativeServiceBuilder) WithImage(name string) *KnativeServiceBuilder {
	if b.Spec.Template.Spec.Containers == nil {
		b.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
	}
	b.Spec.Template.Spec.Containers[0].Image = name
	return b
}

func (b *KnativeServiceBuilder) WithRevision(revisionName string, percent int) *KnativeServiceBuilder {
	b.Spec.Template.Name = revisionName
	nottrue := false
	b.Spec.Traffic = append(b.Spec.Traffic, servingalpha1.TrafficTarget{
		TrafficTarget: servingbeta1.TrafficTarget{
			RevisionName:   revisionName,
			Percent:        percent,
			LatestRevision: &nottrue,
		},
	})
	return b
}

func (b *KnativeServiceBuilder) WithEnv(name string, value string) *KnativeServiceBuilder {
	if b.Spec.Template.Spec.Containers == nil {
		b.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
	}
	container := b.Spec.Template.Spec.Containers[0]
	container.Env = append(container.Env, corev1.EnvVar{Name: name, Value: value})
	return b
}

func CheckServiceReady(obj runtime.Object) (bool, error) {
	service, ok := obj.(*servingalpha1.Service)
	if !ok {
		return false, fmt.Errorf("Expected a Knative service (got: %v)", obj)
	}

	return service.Status.GetCondition(servingalpha1.ServiceConditionReady).IsTrue(), nil
}

func CheckLatestReadyRevisionName(name string) func(obj runtime.Object) (bool, error) {
	return func(obj runtime.Object) (bool, error) {
		service, ok := obj.(*servingalpha1.Service)
		if !ok {
			return false, fmt.Errorf("Expected a Knative service (got: %v)", obj)
		}

		return service.Status.LatestReadyRevisionName == name, nil
	}
}
