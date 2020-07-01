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
package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/iter8-tools/iter8-controller/pkg/analytics/api/v1alpha2"
	iter8v1alpha2 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha2"
)

const (
	destinationKey = "destination_workload"
	namespaceKey   = "destination_service_namespace"
)

// MakeRequest generates request payload to analytics
func MakeRequest(instance *iter8v1alpha2.Experiment) (*v1alpha2.Request, error) {
	return &v1alpha2.Request{}, nil
}

// Invoke sends payload to endpoint and gets response back
func Invoke(log logr.Logger, endpoint string, payload interface{}) (*v1alpha2.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	log.Info("post", "URL", endpoint, "request", string(data))

	raw, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	defer raw.Body.Close()
	body, err := ioutil.ReadAll(raw.Body)

	log.Info("post", "URL", endpoint, "response", string(body))

	if raw.StatusCode >= 400 {
		return nil, fmt.Errorf("%v", string(body))
	}

	var response v1alpha2.Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
