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
package obr

import (
	"github.com/iter8-tools/iter8-controller/pkg/analytics/pbr"
)

const (
	// Strategy is label for strategy
	Strategy string = "optimistic_bayesian_routing"
)

// Service ...
type Service struct {
	pbr.PbrAnalyticsService
}

// GetService ...
func GetService() Service {
	return Service{}
}

// GetPath returns path to be used to access analytics service
// See: https://github.com/iter8-tools/iter8-analytics/blob/master/iter8_analytics/api/analytics/endpoints/analytics.py#L98
func (s Service) GetPath() string {
	return "/api/v1/analytics/canary/optimistic_bayesian_routing"
}
