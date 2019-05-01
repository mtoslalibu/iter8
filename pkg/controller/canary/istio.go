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

package canary

import (
	"context"

	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/knative/pkg/kmeta"
)

func (r *ReconcileCanary) syncIstio(context context.Context, instance *iter8v1alpha1.Canary) (reconcile.Result, error) {
	serviceName := instance.Spec.TargetService.Name
	serviceNamespace := instance.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = instance.Namespace
	}

	// Get k8s service
	service := &corev1.Service{}
	err := r.Get(context, types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, service)
	if err != nil {
		instance.Status.MarkHasNotService("NotFound", "")
		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	instance.Status.MarkHasService()

	// Get deployment list
	deployments := &appsv1.DeploymentList{}
	if err = List(context, &deployments, client.MatchingLabels(service.Spec.Selector)); err != nil {
		// TODO: add new type of status to set unavailable deployments
		instance.Status.MarkHasNotService("NotFound", "")
		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Get current deployment and candidate deployment
	var base, candidate *appsv1.Deployment
	for _, d := deployments.Items {
		if val, ok := d.ObjectMeta.Labels[canaryLabel]; ok {
			if val == "candidate" {
				candidate = d.DeepCopy()
			}else if val == "base" {
				base = d.DeepCopy()
			}
		}
	}

	if base == nil || candidate == nil {
		instance.Status.MarkHasNotService("Base or candidate deployment is missing", "")
		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Start Canary Process
	// Setup Istio Routing Rules

	// Take reviews as an example
	// apiVersion: networking.istio.io/v1alpha3
	// kind: DestinationRule
	// metadata:
	//   name: reviews-canary
	// spec:
	//   host: reviews
	//   subsets:
	//   - name: base
	// 	   labels:
	// 	     iter8.ibm.com/canary: base
	//   - name: candidate
	// 	   labels:
	// 	     iter8.ibm.com/canary: candidate

	


	return reconcile.Result{}, nil
}

func newDestinationRule(canary *iter8v1alpha1.Canary){
	dr := &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: getDestinationRuleName(canary),
			Namespace: serviceNamespace,
			// TODO: add owner references 
		},
		Spec: &v1alpha3.DestinationRuleSpec{
			Host: serviceName,
			Subsets: []*istiov1alpha3.Subset{
				&istiov1alpha3.Subset{
					Name: "base",
					Labels: map[string]string{canaryLabel: "base"},
				},
				&istiov1alpha3.Subset{
					Name: "candidate",
					Labels: map[string]string{canaryLabel: "candidate"},
				},
			},
		},
}
}

func getDestinationRuleName(canary *iter8v1alpha1.Canary) string {
	return canary.Spec.TargetService.Name + "-canary"
}

// apiVersion: networking.istio.io/v1alpha3
	// kind: VirtualService
	// metadata:
	//   name: reviews-canary
	// spec:
	//   hosts:
	// 	- reviews
	//   http:
	//   - route:
	// 		- destination:
	// 			host: reviews
	// 			subset: base
	// 	  	  weight: 50
	// 		- destination:
	// 			host: reviews
	// 			subset: candidate
	// 	  	  weight: 50
	
func newVirtualService(canary *iter8v1alpha1.Canary) *v1alpha3.VirtualService {
	vs := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name: getVirtualServiceName(canary),
			Namespace: canary.Spec.TargetService.Namespace,
			// TODO: add owner references 
		},
		Spec: &v1alpha3.VirtualServiceSpec{
			Gateways: "mesh",
			Hosts: canary.Spec.TargetService.Name ,
			HTTP: &v1alpha3.HTTPRoute{
				Route: v1alpha3.HTTPRouteDestination{
					{
						Destination: v1alpha3.Destination{
							Host: canary.Spec.TargetService.Name ,
							Subset: "base",
						},
						Weight: 100,
					},
					{
						Destination: v1alpha3.Destination{
							Host: canary.Spec.TargetService.Name ,
							Subset: "candidate",
						},
						Weight: 0,
					},		
				},
			},
		},
	}

	return vs
}

func getVirtualServiceName(canary *iter8v1alpha1.Canary) string {
	return canary.Spec.TargetService.Name + "-canary"
}
