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

package experiment

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

func InitEventRecorder(context context.Context, cfg *rest.Config) record.EventRecorder {
	log := Logger(context)
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "FailedToInitKubeClient")
	}

	eb := record.NewBroadcaster()
	eb.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: kubeClient.CoreV1().Events(""),
	})
	er := eb.NewRecorder(
		scheme.Scheme, corev1.EventSource{Component: Iter8Controller})

	return er
}
