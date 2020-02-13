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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/iter8-tools/iter8-controller/pkg/analytics"
	analyticsapi "github.com/iter8-tools/iter8-controller/pkg/analytics/api"
	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
)

func TestMockAnalytics(t *testing.T) {
	logger := Logger(t)
	service := StartAnalytics()
	defer service.Close()
	want := dummyResponse()
	service.AddMock("test-0", want)

	algorithm := analytics.GetAlgorithm("check_and_increment")
	got, err := analytics.Invoke(logger, service.GetURL(), dummyRequest(), algorithm)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("unexpected reponse diff (-want, +got) = %v", diff)
	}
}

func dummyRequest() *analyticsapi.Request {
	return &analyticsapi.Request{
		Name: "test-0",
	}
}
func dummyResponse() analyticsapi.Response {
	return analyticsapi.Response{
		Baseline: analyticsapi.MetricsTraffic{
			TrafficPercentage: 95,
		},
		Candidate: analyticsapi.MetricsTraffic{
			TrafficPercentage: 5,
		},
		Assessment: analyticsapi.Assessment{
			Summary: iter8v1alpha1.Summary{
				AbortExperiment: false,
			},
		},
	}
}
