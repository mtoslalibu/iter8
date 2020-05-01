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

package abstract

import (
	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/targets"
)

type condition string

const (
	conditionUnknown  condition = "unknown"
	conditionDetected condition = "detected"
	conditionDeleted  condition = "deleted"
)

// Status indicates the status of one target
type Status struct {
	role      targets.Role
	condition condition
}

// Targets stores the abstract info for all target items
type Targets struct {
	Namespace        string
	ServiceName      string
	serviceCondition condition
	// a map from deployment name to the status of deployment
	Status map[string]*Status
}

// NewTargets returns a new abstract for targets of an experiment
func NewTargets(instance *iter8v1alpha1.Experiment, namespace string) *Targets {
	ts := make(map[string]*Status)
	ts[instance.Spec.TargetService.Baseline] = &Status{
		role:      targets.RoleBaseline,
		condition: conditionUnknown,
	}
	ts[instance.Spec.TargetService.Candidate] = &Status{
		role:      targets.RoleCandidate,
		condition: conditionUnknown,
	}
	return &Targets{
		Namespace:        namespace,
		ServiceName:      instance.Spec.TargetService.Name,
		serviceCondition: conditionUnknown,
		Status:           ts,
	}
}

func (t *Targets) markTargetFound(name string, found bool) {
	if found {
		t.Status[name].condition = conditionDetected
	} else {
		t.Status[name].condition = conditionDeleted
	}
}

func (t *Targets) markServiceFound(found bool) {
	if found {
		t.serviceCondition = conditionDetected
	} else {
		t.serviceCondition = conditionDeleted
	}
}

func (t *Targets) targetToString(name string) string {
	s, ok := t.Status[name]
	if !ok {
		return ""
	}
	return name + "(" + string(s.role) + ") " + string(s.condition)
}

func (t *Targets) serviceToString() string {
	return t.ServiceName + "(" + string(targets.RoleService) + ") " + string(t.serviceCondition)
}

func (t *Targets) targetRole(name string) targets.Role {
	s, ok := t.Status[name]
	if !ok {
		return ""
	}
	return s.role
}
