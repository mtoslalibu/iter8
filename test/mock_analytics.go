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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
)

const (
	// TestTag identifies the test id
	TestTag = "testid"
)

// AnalyticsService with mock response
type AnalyticsService struct {
	// The service endpoint
	Endpoint string

	// mock maps request to response. The key maps to request.name
	mock map[string]checkandincrement.Response
}

// StartAnalytics starts fake analytics service
func StartAnalytics(port string) *AnalyticsService {
	service := &AnalyticsService{
		mock:     make(map[string]checkandincrement.Response),
		Endpoint: "http://localhost:" + port}

	http.HandleFunc(checkandincrement.Path, service.handler)
	go http.ListenAndServe(":"+port, nil)

	return service
}

func (s *AnalyticsService) handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	var request checkandincrement.Request
	err = json.Unmarshal(b, &request)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(400)
		return
	}

	name := request.Name
	if name == "" {
		w.Write([]byte("missing request name"))
		w.WriteHeader(400)
		return
	}

	response, ok := s.mock[name]
	if !ok {
		w.Write([]byte("missing response for test " + name))
		w.WriteHeader(400)
		return
	}

	raw, err := json.Marshal(response)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("invalid response format %v", err)))
		w.WriteHeader(400)
		return
	}
	w.WriteHeader(200)
	w.Write(raw)
}

// Mock adds response for testid
func (s *AnalyticsService) Mock(name string, response checkandincrement.Response) {
	s.mock[name] = response
}
