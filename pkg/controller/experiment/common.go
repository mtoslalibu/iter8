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
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

const (
	MetricsConfigMap = "iter8config-metrics"
	Iter8Namespace   = "iter8"
)

// Logger gets the logger from the context.
func Logger(ctx context.Context) logr.Logger {
	return ctx.Value(loggerKey).(logr.Logger)
}

func updateGrafanaURL(instance *iter8v1alpha1.Experiment, namespace string) {
	endTsStr := "now"
	if instance.Status.EndTimestamp > 0 {
		endTsStr = strconv.FormatInt(instance.Status.EndTimestamp/int64(time.Millisecond), 10)
	}
	instance.Status.GrafanaURL = instance.Spec.Analysis.GetGrafanaEndpoint() +
		"/d/eXPEaNnZz/iter8-application-metrics?" +
		"var-namespace=" + namespace +
		"&var-service=" + instance.Spec.TargetService.Name +
		"&var-baseline=" + instance.Spec.TargetService.Baseline +
		"&var-candidate=" + instance.Spec.TargetService.Candidate +
		"&from=" + strconv.FormatInt(instance.Status.StartTimestamp/int64(time.Millisecond), 10) +
		"&to=" + endTsStr
}

func markExperimentCompleted(instance *iter8v1alpha1.Experiment) {
	// Clear analysis state
	instance.Status.AnalysisState.Raw = []byte("{}")

	// Update grafana url
	instance.Status.EndTimestamp = metav1.Now().UTC().UnixNano()
	updateGrafanaURL(instance, util.GetServiceNamespace(instance))

	instance.Status.MarkExperimentCompleted()
}

func setLabels(obj runtime.Object, newLabels map[string]string) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	labels := accessor.GetLabels()
	for key, val := range newLabels {
		labels[key] = val
	}
	return nil
}

func validUpdateErr(err error) bool {
	benignMsg := "the object has been modified"
	return strings.Contains(err.Error(), benignMsg)
}
