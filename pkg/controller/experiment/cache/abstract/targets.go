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
)

// TargetRole specifies the role of target
type TargetRole string
type Condition string

const (
	RoleBaseline TargetRole = "baseline"

	RoleCandidate TargetRole = "candidate"

	ConditionUnknown Condition = "unknown"

	ConditionDetected Condition = "detected"

	ConditionDeleted Condition = "deleted"
)

// Status indicates the status of one target
type Status struct {
	role      TargetRole
	condition Condition
}

// Targets stores the abstract info for all target items
type Targets struct {
	Namespace        string
	ServiceName      string
	serviceCondition Condition
	// a map from deployment name to the status of deployment
	Status map[string]*Status
}

// NewTargets returns a new abstract for targets of an experiment
func NewTargets(instance *iter8v1alpha1.Experiment, namespace string) *Targets {
	ts := make(map[string]*Status)
	ts[instance.Spec.TargetService.Baseline] = &Status{
		role:      RoleBaseline,
		condition: ConditionUnknown,
	}
	ts[instance.Spec.TargetService.Candidate] = &Status{
		role:      RoleCandidate,
		condition: ConditionUnknown,
	}
	return &Targets{
		Namespace:        namespace,
		ServiceName:      instance.Spec.TargetService.Name,
		serviceCondition: ConditionUnknown,
		Status:           ts,
	}
}

func (t *Targets) MarkTargetFound(name string, found bool) {
	if found {
		t.Status[name].condition = ConditionDetected
	} else {
		t.Status[name].condition = ConditionDeleted
	}
}

func (t *Targets) MarkServiceFound(found bool) {
	if found {
		t.serviceCondition = ConditionDetected
	} else {
		t.serviceCondition = ConditionDeleted
	}
}
