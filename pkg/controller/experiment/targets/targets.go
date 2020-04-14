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
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

type Role string

const (
	RoleService   Role = "service"
	RoleBaseline  Role = "baseline"
	RoleCandidate Role = "candidate"
)

type Targets struct {
	Service   *corev1.Service
	Baseline  *appsv1.Deployment
	Candidate *appsv1.Deployment
}

func InitTargets() *Targets {
	return &Targets{
		Service:   &corev1.Service{},
		Baseline:  &appsv1.Deployment{},
		Candidate: &appsv1.Deployment{},
	}
}

func (t *Targets) Cleanup(context context.Context, instance *iter8v1alpha1.Experiment, client client.Client) {
	if instance.Spec.CleanUp == iter8v1alpha1.CleanUpDelete {
		var stableTarget Role
		if instance.Succeeded() {
			switch instance.Spec.TrafficControl.GetOnSuccess() {
			case "baseline":
				stableTarget = RoleBaseline
			case "candidate":
				stableTarget = RoleCandidate
			}
		} else {
			stableTarget = RoleBaseline
		}

		switch stableTarget {
		case RoleBaseline:
			if err := client.Delete(context, t.Candidate); err != nil {
				util.Logger(context).Error(err, "Delete Candidate", "")
			}
			instance.Status.TrafficSplit.Baseline = 100
			instance.Status.TrafficSplit.Candidate = 0
		case RoleCandidate:
			if err := client.Delete(context, t.Baseline); err != nil {
				util.Logger(context).Error(err, "Delete Baseline", "")
			}
			instance.Status.TrafficSplit.Baseline = 0
			instance.Status.TrafficSplit.Candidate = 100
		}
	}
}
