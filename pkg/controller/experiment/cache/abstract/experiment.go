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
	MarkTargetFound(name string, found bool)
	MarkServiceFound(found bool)
}

type Snapshot interface {
	Terminate() bool
	GetTerminateStatus() string
	GetDeletedRole() string
}

var _ ExperimentInterface = &Experiment{}
var _ Snapshot = &Experiment{}

// Experiment includes abstract info for one Experiment
type Experiment struct {
	Namespace       string
	TargetsAbstract *Targets

	terminate       bool
	terminateStatus string
	deletedRole     string
}

func NewExperiment(instance *iter8v1alpha1.Experiment, targetNamespace string) *Experiment {
	return &Experiment{
		Namespace:       instance.Namespace,
		TargetsAbstract: NewTargets(instance, targetNamespace),
	}
}

func (e *Experiment) Terminate() bool {
	return e.terminate
}

func (e *Experiment) GetTerminateStatus() string {
	return e.terminateStatus
}

func (e *Experiment) GetDeletedRole() string {
	return e.deletedRole
}

func (e *Experiment) MarkTargetFound(name string, found bool) {
	e.TargetsAbstract.markTargetFound(name, found)
	if found == false {
		e.terminate = true
		e.terminateStatus = e.TargetsAbstract.targetToString(name)
		e.deletedRole = e.TargetsAbstract.targetRole(name)
	}
}

func (e *Experiment) MarkServiceFound(found bool) {
	e.TargetsAbstract.markServiceFound(found)
	if found == false {
		e.terminate = true
		e.terminateStatus = e.TargetsAbstract.serviceToString()
		e.deletedRole = string(RoleService)
	}
}

func (e *Experiment) GetSnapshot() Snapshot {
	out := &Experiment{}
	*out = *e
	return out
}
