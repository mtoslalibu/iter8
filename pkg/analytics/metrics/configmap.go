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
package metrics

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
)

const (
	metricsConfigMap = "iter8config-metrics"
	defaultNamespace = "iter8"
)

// Metrics list of Metric
type Metrics []Metric

// Metric structure of cm/iter8_metric
type Metric struct {
	Name               string `yaml:"name"`
	IsCounter          bool   `yaml:"is_counter"`
	AbsentValue        string `yaml:"absent_value"`
	SampleSizeTemplate string `yaml:"sample_size_query_template"`
}

// Read reads content in the configmap into the experiment instance
func Read(context context.Context, c client.Client, instance *iter8v1alpha1.Experiment) error {
	cm := &corev1.ConfigMap{}
	err := c.Get(context, types.NamespacedName{Name: metricsConfigMap, Namespace: iter8Namespace()}, cm)
	if err != nil {
		if err = c.Get(context, types.NamespacedName{Name: metricsConfigMap, Namespace: instance.GetNamespace()}, cm); err != nil {
			return fmt.Errorf("MetricsConfigMapNotFound: %s", err)
		}
	}

	data := cm.Data
	var templates map[string]string
	metrics := Metrics{}

	err = yaml.Unmarshal([]byte(data["query_templates"]), &templates)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(data["metrics"]), &metrics)
	if err != nil {
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

func iter8Namespace() string {
	retVal := defaultNamespace
	if namespace := os.Getenv("SERVICE_NAMESPACE"); namespace != "" {
		retVal = namespace
	}
	return retVal
}
