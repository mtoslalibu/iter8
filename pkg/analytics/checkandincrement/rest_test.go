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
	"testing"
)

type testCase struct {
	endpoint string
	request  Request
}

func TestInvoke(t *testing.T) {
	table := []testCase{
		testCase{
			endpoint: "localhost:5555",
			request: Request{
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
							Type:       SuccessCriterionDelta,
							Value:      0.02,
						},
					},
				},
				LastState: map[string]string{},
			},
		},
	}
	for _, tc := range table {
		_, err := Invoke(tc.endpoint, tc.request)
		if err != nil {
			t.Error(err)
		}
	}

}
