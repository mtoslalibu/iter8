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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
)

const (
	MetricsConfigMap = "iter8config-metrics"
	Iter8Namespace   = "iter8"
)

// Logger gets the logger from the context.
func Logger(ctx context.Context) logr.Logger {
	return ctx.Value(loggerKey).(logr.Logger)
}

func addFinalizerIfAbsent(context context.Context, c client.Client, instance *iter8v1alpha1.Experiment, fName string) (err error) {
	for _, finalizer := range instance.ObjectMeta.GetFinalizers() {
		if finalizer == fName {
			return
		}
	}

	instance.SetFinalizers(append(instance.GetFinalizers(), Finalizer))
	if err = c.Update(context, instance); err != nil {
		Logger(context).Info("setting finalizer failed. (retrying)", "error", err)
	}

	return
}

func removeFinalizer(context context.Context, c client.Client, instance *iter8v1alpha1.Experiment, fName string) (err error) {
	finalizers := make([]string, 0)
	for _, f := range instance.GetFinalizers() {
		if f != fName {
			finalizers = append(finalizers, f)
		}
	}
	instance.SetFinalizers(finalizers)
	if err = c.Update(context, instance); err != nil {
		Logger(context).Info("setting finalizer failed. (retrying)", "error", err)
	}

	Logger(context).Info("FinalizerRemoved")
	return
}

func updateGrafanaURL(instance *iter8v1alpha1.Experiment, namespace string) {
	endTs := instance.Status.EndTimestamp
	if endTs == "" {
		endTs = "now"
	}
	instance.Status.GrafanaURL = instance.Spec.Analysis.GetGrafanaEndpoint() +
		"/d/eXPEaNnZz/iter8-application-metrics?" +
		"var-namespace=" + namespace +
		"&var-service=" + instance.Spec.TargetService.Name +
		"&var-baseline=" + instance.Spec.TargetService.Baseline +
		"&var-candidate=" + instance.Spec.TargetService.Candidate +
		"&from=" + instance.Status.StartTimestamp +
		"&to=" + endTs
}

func successMsg(instance *iter8v1alpha1.Experiment) string {
	if instance.Action == iter8v1alpha1.ActionOverrideSuccess {
		return "Override Success"
	} else if instance.Status.AssessmentSummary.AllSuccessCriteriaMet {
		return "All Success Criteria Were Met"
	} else {
		return "Last Iteration Was Completed"
	}
}

func failureMsg(instance *iter8v1alpha1.Experiment) string {
	if instance.Action == iter8v1alpha1.ActionOverrideFailure {
		return "Override Failure"
	} else if !instance.Status.AssessmentSummary.AllSuccessCriteriaMet {
		return "Not All Success Criteria Met"
	} else {
		// Should not be reached
		return "Unexpected Condition"
	}
}

// Metrics list of Metric
type Metrics []Metric

// Metric structure of cm/iter8_metric
type Metric struct {
	Name               string `yaml:"name"`
	IsCounter          bool   `yaml:"is_counter"`
	AbsentValue        string `yaml:"absent_value"`
	SampleSizeTemplate string `yaml:"sample_size_query_template"`
}

func readMetrics(context context.Context, c client.Client, instance *iter8v1alpha1.Experiment) error {
	log := Logger(context)
	cm := &corev1.ConfigMap{}
	err := c.Get(context, types.NamespacedName{Name: MetricsConfigMap, Namespace: Iter8Namespace}, cm)
	if err != nil {
		if err = c.Get(context, types.NamespacedName{Name: MetricsConfigMap, Namespace: instance.GetNamespace()}, cm); err != nil {
			log.Info("MetricsConfigMapNotFound")
			return err
		}
	}

	data := cm.Data
	var templates map[string]string
	metrics := Metrics{}

	err = yaml.Unmarshal([]byte(data["query_templates"]), &templates)
	if err != nil {
		log.Error(err, "FailToReadYaml", "query_templates", data["query_templates"])
		return err
	}

	err = yaml.Unmarshal([]byte(data["metrics"]), &metrics)
	if err != nil {
		log.Error(err, "FailToReadYaml", "metrics", data["metrics"])
		return err
	}

	instance.Metrics = make(map[string]iter8v1alpha1.ExperimentMetric)
	criteria := make(map[string]bool)

	for _, criterion := range instance.Spec.Analysis.SuccessCriteria {
		criteria[criterion.MetricName] = true
	}

	for _, metric := range metrics {
		if !criteria[metric.Name] {
			continue
		}

		m := iter8v1alpha1.ExperimentMetric{
			IsCounter:   metric.IsCounter,
			AbsentValue: metric.AbsentValue,
		}
		qTpl, ok := templates[metric.Name]
		if !ok {
			return fmt.Errorf("FailToReadQueryTemplateForMetric %s", metric.Name)
		}
		sTpl, ok := templates[metric.SampleSizeTemplate]
		if !ok {
			return fmt.Errorf("FailToReadSampleSizeTemplateForMetric %s", metric.Name)
		}
		m.QueryTemplate = qTpl
		m.SampleSizeTemplate = sTpl
		instance.Metrics[metric.Name] = m
	}

	return c.Update(context, instance)
}

func (r *ReconcileExperiment) executeStatusUpdate(ctx context.Context, instance *iter8v1alpha1.Experiment) (err error) {
	trial := 3
	for trial > 0 {
		if err = r.Status().Update(ctx, instance); err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		trial--
	}
	return
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
