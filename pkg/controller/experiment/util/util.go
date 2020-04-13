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
	"strings"

	"github.com/go-logr/logr"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/cache/abstract"
)

type loggerKeyType string

// Logger gets the logger from the context.
func Logger(ctx context.Context) logr.Logger {
	return ctx.Value(LoggerKey).(logr.Logger)
}

const (
	AbstractKey = "experimentAbstract"
	LoggerKey   = loggerKeyType("logger")
)

func ExperimentAbstract(ctx context.Context) abstract.Snapshot {
	return ctx.Value(AbstractKey).(abstract.Snapshot)
}

func GetServiceNamespace(instance *iter8v1alpha1.Experiment) string {
	serviceNamespace := instance.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = instance.Namespace
	}
	return serviceNamespace
}

func GetStableTarget(context context.Context, instance *iter8v1alpha1.Experiment) string {
	out := ""

	if instance.Succeeded() {
		out = instance.Spec.TrafficControl.GetOnSuccess()
	} else {
		out = "baseline"
	}

	return out
}

func EqualHost(host1, ns1, host2, ns2 string) bool {
	if host1 == host2 ||
		host1 == host2+"."+ns2+".svc.cluster.local" ||
		host1+"."+ns1+".svc.cluster.local" == host2 {
		return true
	}
	return false
}

func ValidUpdateErr(err error) bool {
	benignMsg := "the object has been modified"
	return strings.Contains(err.Error(), benignMsg)
}
