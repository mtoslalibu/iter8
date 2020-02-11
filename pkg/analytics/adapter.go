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
	"github.com/iter8-tools/iter8-controller/pkg/analytics/algorithm"
	"github.com/iter8-tools/iter8-controller/pkg/analytics/algorithm/check_and_increment"
	"github.com/iter8-tools/iter8-controller/pkg/analytics/algorithm/epsilongreedy"
	"github.com/iter8-tools/iter8-controller/pkg/analytics/algorithm/obr"
	"github.com/iter8-tools/iter8-controller/pkg/analytics/algorithm/pbr"
)

func GetAlgorithm(strategy string) algorithm.Interface {
	switch strategy {
	case check_and_increment.Strategy:
		return check_and_increment.Impl{}
	case epsilongreedy.Strategy:
		return epsilongreedy.Impl{}
	case obr.Strategy:
		return obr.Impl{}
	case pbr.Strategy:
		return pbr.Impl{}
	}
	return nil
}
