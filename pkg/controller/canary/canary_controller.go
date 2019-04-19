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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"

	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

var log = logf.Log.WithName("canary-controller")

const (
	canaryLabel = "iter8.ibm.com/canary"
)

// Add creates a new Canary Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCanary{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("canary-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Canary
	err = c.Watch(&source.Kind{Type: &iter8v1alpha1.Canary{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch changes to Knative services
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			canary := a.Meta.GetLabels()[canaryLabel]
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      canary,
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		})

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if _, ok := e.MetaOld.GetLabels()[canaryLabel]; !ok {
				return false
			}
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Meta.GetLabels()[canaryLabel]
			return ok
		},
	}

	err = c.Watch(&source.Kind{Type: &servingv1alpha1.Service{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		p)

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCanary{}

// ReconcileCanary reconciles a Canary object
type ReconcileCanary struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Canary object and makes changes based on the state read
// and what is in the Canary.Spec
// +kubebuilder:rbac:groups=iter8.ibm.com,resources=canaries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iter8.ibm.com,resources=canaries/status,verbs=get;update;patch
func (r *ReconcileCanary) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	context := context.Background()

	// Fetch the Canary instance
	instance := &iter8v1alpha1.Canary{}
	err := r.Get(context, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check whether object has been deleted
	if instance.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	if instance.Generation == instance.Status.ObservedGeneration {
		log.Info("synchronized", "namespace", instance.Namespace, "name", instance.Name)
		return reconcile.Result{}, nil
	}

	log.Info("reconciling", "namespace", instance.Namespace, "name", instance.Name)

	instance.Status.InitializeConditions()

	// TODO: not sure why this is needed
	if instance.Status.LastIncrementTime.IsZero() {
		instance.Status.LastIncrementTime = metav1.NewTime(time.Unix(0, 0))
	}

	// Get Knative service
	serviceName := instance.Spec.TargetService.Name
	serviceNamespace := instance.Spec.TargetService.Namespace
	if serviceNamespace == "" {
		serviceNamespace = instance.Namespace
	}

	kservice := &servingv1alpha1.Service{}
	err = r.Get(context, types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, kservice)
	if err != nil {
		instance.Status.MarkHasNotService("NotFound", "")
		err = r.Status().Update(context, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if kservice.Spec.DeprecatedPinned != nil {
		instance.Status.MarkHasNotService("DeprecatedPinnedNotSupported", "")
		err = r.Status().Update(context, instance)
		return reconcile.Result{}, err
	}

	// link service to this canary. Only one canary can control a service
	labels := kservice.GetLabels()
	if labels != nil && labels[canaryLabel] != instance.GetName() {
		instance.Status.MarkHasNotService("ExistingCanary", "service is already controlled by %v", labels[canaryLabel])
		err = r.Status().Update(context, instance)
		return reconcile.Result{}, err
	}

	if labels == nil {
		labels = make(map[string]string)
	}

	if _, ok := labels[canaryLabel]; !ok {
		labels[canaryLabel] = instance.GetName()
		kservice.SetLabels(labels)
		if err = r.Update(context, kservice); err != nil {
			return reconcile.Result{}, err
		}
	}

	instance.Status.MarkHasService()

	// Check mode is set to 'release'. If not, change it
	if kservice.Spec.Release == nil {
		// TODO: maybe should check if equal to LastedCreatedRevisionName?
		kservice.Spec.Release = &servingv1alpha1.ReleaseType{
			Revisions: []string{
				kservice.Status.LatestReadyRevisionName,
			},
			RolloutPercent: 0,
			Configuration:  kservice.Spec.RunLatest.Configuration,
		}
		kservice.Spec.RunLatest = nil
		err := r.Update(context, kservice)
		if err != nil {
			instance.Status.MarkMinimumRevisionNotAvailable("NotReleaseMode", "%v", err)
			err = r.Status().Update(context, instance)
			return reconcile.Result{}, err
		}
	}

	// Release must have at least 2 revisions
	if len(kservice.Spec.Release.Revisions) < 2 {
		instance.Status.MarkMinimumRevisionNotAvailable("NotEnoughRevisions", "")
		err := r.Status().Update(context, instance)
		return reconcile.Result{}, err
	}

	instance.Status.MarkMinimumRevisionAvailable()

	// Increment traffic when applicable
	traffic := instance.Spec.TrafficControl
	release := kservice.Spec.Release
	now := time.Now()
	interval := traffic.GetInterval()

	if release.RolloutPercent < traffic.GetMaxTrafficPercent() &&
		now.After(instance.Status.LastIncrementTime.Add(interval)) {

		// Due for increment
		release.RolloutPercent += traffic.GetStepSize()
		if release.RolloutPercent > traffic.GetMaxTrafficPercent() {
			release.RolloutPercent = traffic.GetMaxTrafficPercent()
		}

		err := r.Update(context, kservice)
		if err != nil {
			return reconcile.Result{}, err
		}

		instance.Status.LastIncrementTime = metav1.NewTime(now)
	}

	result := reconcile.Result{}
	if release.RolloutPercent == traffic.GetMaxTrafficPercent() {
		// Rollout done.
		instance.Status.ObservedGeneration = instance.Generation
		instance.Status.Progressing = false
	} else {
		instance.Status.Progressing = true
		result.RequeueAfter = interval
	}

	err = r.Status().Update(context, instance)
	return result, err
}
