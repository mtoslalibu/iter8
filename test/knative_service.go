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
	"context"
	"fmt"

	servingalpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KnativeServiceBuilder builds Knative service object
type KnativeServiceBuilder servingalpha1.Service

// NewKnativeService creates a default Knative service
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
	}
	s.SetDefaults(context.TODO())
	return (*KnativeServiceBuilder)(s)
}

// Build the Knative Service object
func (b *KnativeServiceBuilder) Build() *servingalpha1.Service {
	return (*servingalpha1.Service)(b)
}

func (b *KnativeServiceBuilder) WithReleaseImage(image string) *KnativeServiceBuilder {
	if b.Spec.Release == nil {
		b.Spec.Release = &servingalpha1.ReleaseType{}
	}
	b.Spec.Release.Configuration.RevisionTemplate.Spec.Container.Image = image
	return b
}

func (b *KnativeServiceBuilder) WithReleaseRevisions(revisions []string) *KnativeServiceBuilder {
	if b.Spec.Release == nil {
		b.Spec.Release = &servingalpha1.ReleaseType{}
	}
	b.Spec.Release.Revisions = revisions
	return b
}

func (b *KnativeServiceBuilder) WithReleaseEnv(name string, value string) *KnativeServiceBuilder {
	if b.Spec.Release == nil {
		b.Spec.Release = &servingalpha1.ReleaseType{}
	}
	container := b.Spec.Release.Configuration.RevisionTemplate.Spec.Container
	container.Env = append(container.Env, corev1.EnvVar{Name: name, Value: value})
	return b
}

func (b *KnativeServiceBuilder) WithLatestImage(image string) *KnativeServiceBuilder {
	if b.Spec.RunLatest == nil {
		b.Spec.RunLatest = &servingalpha1.RunLatestType{}
	}
	b.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Image = image
	return b
}

func GetLatestRevision(ctx context.Context, cl client.Client, name string, namespace string) (string, error) {
	obj := NewKnativeService(name, namespace).Build()

	err := cl.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	if err != nil {
		return "", err
	}

	return obj.Status.LatestReadyRevisionName, nil
}

func CheckServiceReady(obj runtime.Object) (bool, error) {
	service, ok := obj.(*servingalpha1.Service)
	if !ok {
		return false, fmt.Errorf("Expected a Knative service (got: %v)", obj)
	}

	return service.Status.GetCondition(servingalpha1.ServiceConditionReady).IsTrue(), nil
}
