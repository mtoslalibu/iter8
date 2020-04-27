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

package v1alpha2

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type Experiment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec    ExperimentSpec    `json:"spec,omitempty"`
	Status  ExperimentStatus  `json:"status,omitempty"`
	Metrics ExperimentMetrics `json:"metrics,omitempty"`
	// Action provides user an option to take action in the experiment
	// pause: pause the progress of experiment
	// resume: resume the experiment
	// override_failure: force the experiment to failure status
	// override_success: force the experiment to success status
	// +optional.
	//+kubebuilder:validation:Enum={pause,resume,override_failure,override_success}
	Action ExperimentAction `json:"action,omitempty"`
}

// ExperimentList contains a list of Experiment
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExperimentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Experiment `json:"items"`
}

// ExperimentSpec defines the desired state of Experiment
type ExperimentSpec struct {
	// TargetService is a reference to an object to use as target service
	TargetService TargetService `json:"service"`

	// RoutingReference provides references to routing rules set by users
	// +optional
	RoutingReference *corev1.ObjectReference `json:"routingReference,omitempty"`
}
