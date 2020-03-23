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

package targets

import (
	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

// TargetRole specifies the role of target
type TargetRole string

const (
	RoleBaseline TargetRole = "baseline"

	RoleCandidate TargetRole = "candidate"
)

// Status indicates the status of one target
type Status struct {
	role    TargetRole
	existed bool
}

// Abstract stores the abstract info for all target items
type Abstract struct {
	Namespace      string
	ServiceName    string
	serviceExisted bool
	// a map from deployment name to the status of deployment
	Status map[string]*Status
}

// NewAbstract returns a new abstract for targets of an experiment
func NewAbstract(instance *iter8v1alpha1.Experiment) *Abstract {
	ts := make(map[string]*Status)
	ts[instance.Spec.TargetService.Baseline] = &Status{
		role: RoleBaseline,
	}
	ts[instance.Spec.TargetService.Candidate] = &Status{
		role: RoleCandidate,
	}
	return &Abstract{
		Namespace:   util.GetServiceNamespace(instance),
		ServiceName: instance.Spec.TargetService.Name,
		Status:      ts,
	}
}
