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
package checkandincrement

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	iter8v1alpha1 "github.com/iter8.tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type testCase struct {
	endpoint string
	request  Request
	response Response
}

func TestInvoke(t *testing.T) {
	req1 := Request{
		Baseline: Window{
			StartTime: "2019-05-01T18:53:35.163Z",
			EndTime:   "2019-05-01T18:53:35.163Z",
			Tags: map[string]string{
				"destination_service_name": "reviews-v2",
			},
		},
		Canary: Window{
			StartTime: "2019-05-01T18:53:35.163Z",
			EndTime:   "2019-05-01T18:53:35.163Z",
			Tags: map[string]string{
				"destination_service_name": "reviews-v2",
			},
		},
		TrafficControl: TrafficControl{
			SuccessCriteria: []SuccessCriterion{
				SuccessCriterion{
					MetricName: "iter8_latency",
					Type:       iter8v1alpha1.ToleranceTypeDelta,
					Value:      0.02,
				},
			},
		},
		LastState: map[string]string{},
	}

	resp1 := Response{
		Baseline: MetricsTraffic{
			TrafficPercentage: 95,
		},
		Canary: MetricsTraffic{
			TrafficPercentage: 5,
		},
		Assessment: Assessment{
			Summary: iter8v1alpha1.Summary{
				AbortExperiment: false,
			},
		},
	}
	rawResp1, err := json.Marshal(resp1)
	if err != nil {
		t.Error(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := ioutil.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(400)
			return
		}

		rw.WriteHeader(200)
		rw.Write(rawResp1)
	}))
	// Close the server when test finishes
	defer server.Close()

	table := []testCase{
		testCase{
			endpoint: server.URL,
			request:  req1,
			response: resp1,
		},
	}

	log.SetLogger(log.ZapLogger(false))
	logger := log.Log.WithName("entrypoint")
	for _, tc := range table {
		response, err := Invoke(logger, tc.endpoint, &tc.request)
		if err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(*response, resp1); diff != "" {
			t.Error(diff)
		}
	}
}
