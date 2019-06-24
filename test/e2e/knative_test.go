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
package e2e

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cai "github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"

	"github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.ibm.com/istio-research/iter8-controller/test"
)

// TestKnativeExperiment tests various experiment scenarios on Knative platform
func TestKnativeExperiment(t *testing.T) {
	service := test.StartAnalytics()
	defer service.Close()
	testCases := map[string]testCase{
		"missingService": testCase{
			object:      getDoNotExistExperiment(),
			wantResults: []runtime.Object{getDoNotExistExperimentReconciled()},
		},
		"deleteExperimentAfterCompleted": testCase{
			mocks: map[string]cai.Response{
				"stock-1": test.GetDefaultMockResponse(),
			},
			initObjects: []runtime.Object{
				getBaseStockService("stock-1"),
			},
			object:    getFastExperimentForService("stock-1", "stock-1", service.GetURL()),
			preHook:   newStockServiceRevision("stock-1"),
			wantState: test.CheckExperimentFinished,
			frozenObjects: []runtime.Object{
				getBaseStockService("stock-1"),
			},
			postHook: test.DeleteExperiment("stock-1", Flags.Namespace),
		},
	}

	runTestCases(t, service, testCases)
}

func getDoNotExistExperiment() *v1alpha1.Experiment {
	return test.NewExperiment("experiment-missing-service", Flags.Namespace).
		WithKNativeService("doesnotexist").
		Build()
}

func getFastExperimentForService(name string, serviceName string, analyticsHost string) *v1alpha1.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKNativeService(serviceName).
		WithAnalyticsHost(analyticsHost).
		Build()
	onesec := "1s"
	one := 1
	experiment.Spec.TargetService.Baseline = serviceName + "-one"
	experiment.Spec.TargetService.Candidate = serviceName + "-two"
	experiment.Spec.TrafficControl.Interval = &onesec
	experiment.Spec.TrafficControl.MaxIterations = &one
	return experiment
}

func getDoNotExistExperimentReconciled() *v1alpha1.Experiment {
	experiment := getDoNotExistExperiment()
	experiment.Status.MarkExperimentNotCompleted("Progressing", "")
	experiment.Status.MarkHasNotService("NotFound", "")
	return experiment
}

func getBaseStockService(name string) runtime.Object {
	return test.NewKnativeService(name, Flags.Namespace).
		WithImage(test.StockImageName).
		Build()
}

func newStockServiceRevision(name string) test.Hook {
	return func(ctx context.Context, cl client.Client) error {
		ksvc := test.NewKnativeService(name, Flags.Namespace).Build()

		err := test.WaitForState(ctx, cl, ksvc, test.CheckServiceReady)
		if err != nil {
			return err
		}

		(*test.KnativeServiceBuilder)(ksvc).
			WithEnv("RESOURCE", "share").
			WithRevision("two")

		if err := cl.Update(ctx, ksvc); err != nil {
			return errors.Wrapf(err, "Cannot update service %s", name)
		}

		return test.WaitForState(ctx, cl, ksvc, test.CheckLatestReadyRevisionName(name+"-two"))
	}
}
