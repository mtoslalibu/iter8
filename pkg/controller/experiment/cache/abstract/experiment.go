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

type ExperimentInterface interface {
	Terminate() bool
	GetDeletedTarget() string
	MarkTargetFound(name string, found bool)
	MarkServiceFound(found bool)
}

var _ ExperimentInterface = &Experiment{}

// Experiment includes abstract info for one Experiment
type Experiment struct {
	namespace       string
	TargetsAbstract *Targets
	terminate       bool
}

func NewExperiment(instance *iter8v1alpha1.Experiment, targetNamespace string) *Experiment {
	return &Experiment{
		namespace:       instance.Namespace,
		TargetsAbstract: NewTargets(instance, targetNamespace),
	}
}

func (e *Experiment) Terminate() bool {
	return e.terminate
}

func (e *Experiment) GetDeletedTarget() string {
	if e.TargetsAbstract.serviceCondition == ConditionDeleted {
		return string(RoleService)
	}

	for _, status := range e.TargetsAbstract.Status {
		if status.condition == ConditionDeleted {
			return string(status.role)
		}
	}

	return ""
}

func (e *Experiment) MarkTargetFound(name string, found bool) {
	if found == false {
		e.terminate = true
	}
	e.TargetsAbstract.MarkTargetFound(name, found)
}

func (e *Experiment) MarkServiceFound(found bool) {
	if found == false {
		e.terminate = true
	}
	e.TargetsAbstract.MarkServiceFound(found)
}
