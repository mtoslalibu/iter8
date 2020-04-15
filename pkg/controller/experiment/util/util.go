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
package util

import (
	"context"

	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/cache/abstract"
)

const (
	AbstractKey = "experimentAbstract"
)

func ExperimentAbstract(ctx context.Context) abstract.ExperimentInterface {
	return ctx.Value(AbstractKey).(abstract.ExperimentInterface)
}

func GetServiceNamespace(instance *iter8v1alpha1.Experiment) string {
	serviceNamespace := instance.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = instance.Namespace
	}
	return serviceNamespace
}

func DeleteObjects(context context.Context, client client.Client, objs ...runtime.Object) error {
	for _, obj := range objs {
		if err := client.Delete(context, obj); err != nil {
			return err
		}
	}
	return nil
}

func EqualHost(host1, ns1, host2, ns2 string) bool {
	if host1 == host2 ||
		host1 == host2+"."+ns2+".svc.cluster.local" ||
		host1+"."+ns1+".svc.cluster.local" == host2 {
		return true
	}
	return false
}
