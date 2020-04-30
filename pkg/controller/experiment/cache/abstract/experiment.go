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

type snapshotKeyType string
type action string

const (
	SnapshotKey = snapshotKeyType("experimentAbstract")

	actionTerminate action = "terminate"
	actionResume    action = "resume"
)

type ExperimentInterface interface {
	MarkTargetFound(name string, found bool)
	MarkServiceFound(found bool)
}

type Snapshot interface {
	Terminate() bool
	Resume() bool
	GetTerminateStatus() string
	GetDeletedRole() targets.Role
}

var _ ExperimentInterface = &Experiment{}
var _ Snapshot = &Experiment{}

// Experiment includes abstract info for one Experiment
type Experiment struct {
	Namespace       string
	TargetsAbstract *Targets

	action          action
	terminateStatus string
	deletedRole     targets.Role
}

func NewExperiment(instance *iter8v1alpha1.Experiment, targetNamespace string) *Experiment {
	return &Experiment{
		Namespace:       instance.Namespace,
		TargetsAbstract: NewTargets(instance, targetNamespace),
	}
}

func (e *Experiment) Terminate() bool {
	return e.action == actionTerminate
}

func (e *Experiment) Resume() bool {
	return e.action == actionResume
}

func (e *Experiment) clear() {
	e.action = ""
	e.terminateStatus = ""
	e.deletedRole = ""
}

func (e *Experiment) GetTerminateStatus() string {
	return e.terminateStatus
}

func (e *Experiment) GetDeletedRole() targets.Role {
	return e.deletedRole
}

func (e *Experiment) MarkTargetFound(name string, found bool) {
	e.TargetsAbstract.markTargetFound(name, found)
	if found == false {
		e.action = actionTerminate
		e.terminateStatus = e.TargetsAbstract.targetToString(name)
		e.deletedRole = e.TargetsAbstract.targetRole(name)
	} else {
		e.action = actionResume
	}
}

func (e *Experiment) MarkServiceFound(found bool) {
	e.TargetsAbstract.markServiceFound(found)
	if found == false {
		e.action = actionTerminate
		e.terminateStatus = e.TargetsAbstract.serviceToString()
		e.deletedRole = targets.RoleService
	} else {
		e.action = actionResume
	}
}

func (e *Experiment) GetSnapshot() Snapshot {
	out := &Experiment{}
	*out = *e
	e.clear()
	return out
}
