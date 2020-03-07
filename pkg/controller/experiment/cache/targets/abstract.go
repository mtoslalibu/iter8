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
	"sync"
)

// TargetRole specifies the role of target
type TargetRole string

const (
	RoleBaseline TargetRole = "baseline"

	RoleCandidate TargetRole = "candidate"
)

// TargetStatus indicates the status of one target
type TargetStatus struct {
	role    TargetRole
	existed bool
}

// TargetsAbstract stores the abstract info for all target items
type TargetsAbstract struct {
	Namespace      string
	ServiceName    string
	serviceExisted bool
	// a map from deployment name to the status of deployment
	targetsStatus map[string]*TargetStatus
	// the lock for reading/modifying the status map
	m sync.RWMutex
}

func (t *TargetsAbstract) updateServiceStatus(found bool) {
	t.serviceExisted = found
}

func (t *TargetsAbstract) targetExisted(name string) bool {
	t.m.Lock()
	defer t.m.Unlock()
	_, ok := t.targetsStatus[name]
	return ok
}

func (t *TargetsAbstract) updateTargetStatus(name string, found bool) {
	t.m.Lock()
	defer t.m.Unlock()
	t.targetsStatus[name].existed = found
}
