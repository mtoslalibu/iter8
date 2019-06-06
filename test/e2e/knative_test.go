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

	"k8s.io/apimachinery/pkg/runtime"

	"github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.ibm.com/istio-research/iter8-controller/test"
)

// TestKnativeExperiment tests various experiment scenarios on Knative platform
func TestKnativeExperiment(t *testing.T) {
	//logger := test.Logger(t)
	service := test.StartAnalytics()
	defer service.Close()

	testCases := map[string]testCase{
		"missingservice": testCase{
			object:      getDoNotExistExperiment(),
			wantResults: []runtime.Object{getDoNotExistExperimentReconciled()},
		},
	}
	client := GetClient()
	ctx := context.Background()

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			if err := tc.createObject(ctx, client); err != nil {
				t.Fatal(err)
			}

			if err := tc.checkHasResults(ctx, client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func getDoNotExistExperiment() *v1alpha1.Experiment {
	return test.NewExperiment("experiment-missing-service", Flags.Namespace).
		WithKNativeService("doesnotexist").
		Build()
}

func getDoNotExistExperimentReconciled() *v1alpha1.Experiment {
	experiment := getDoNotExistExperiment()
	experiment.Status.MarkExperimentNotCompleted("Progressing", "")
	experiment.Status.MarkHasNotService("NotFound", "")
	return experiment
}
