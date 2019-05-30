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

	"github.com/go-logr/logr"
	"go.uber.org/zap"
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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

var log = logf.Log.WithName("canary-controller")

type loggerKeyType string

const (
	canaryLabel = "iter8.ibm.com/canary"

	KubernetesService      = "v1"
	KnativeServiceV1Alpha1 = "serving.knative.dev/v1alpha1"

	Finalizer = "finalizer.iter8.ibm.com"
	loggerKey = loggerKeyType("logger")
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

	// Watch for Knative services changes
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

	err = c.Watch(&source.Kind{Type: &servingv1alpha1.Service{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		p)

	if err != nil {
		log.Info("NoKnativeServingWatch", zap.Error(err))
	}

	// Watch for k8s deployment updates
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		p)

	if err != nil {
		// Just log error.
		log.Info("NoDeployementWatch", zap.Error(err))
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
	ctx := context.Background()

	// Fetch the Canary instance
	instance := &iter8v1alpha1.Canary{}
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log := log.WithValues("namespace", instance.Namespace, "name", instance.Name)
	ctx = context.WithValue(ctx, loggerKey, log)

	// Add finalizer to the canary object
	if err = addFinalizerIfAbsent(ctx, r, instance, Finalizer); err != nil {
		return reconcile.Result{}, err
	}

	// Check whether object has been deleted
	if instance.DeletionTimestamp != nil {
		return r.finalize(ctx, instance)
	}

	// // Stop right here if the experiment is completed.
	completed := instance.Status.GetCondition(iter8v1alpha1.CanaryConditionExperimentCompleted)
	if completed != nil && completed.Status == corev1.ConditionTrue {
		log.Info("RolloutCompleted", "Use a different name for canary object to trigger a new experiment", "")
		return reconcile.Result{}, nil
	}

	log.Info("reconciling")

	instance.Status.InitializeConditions()

	// TODO: not sure why this is needed
	if instance.Status.LastIncrementTime.IsZero() {
		instance.Status.LastIncrementTime = metav1.NewTime(time.Unix(0, 0))
	}

	if instance.Status.AnalysisState.Raw == nil {
		instance.Status.AnalysisState.Raw = []byte("{}")
	}

	creationts := instance.ObjectMeta.GetCreationTimestamp()
	now := metav1.Now()
	if !creationts.Before(&now) {
		// Delay experiment by 1 sec
		return reconcile.Result{RequeueAfter: time.Second}, nil
	}

	apiVersion := instance.Spec.TargetService.APIVersion

	switch apiVersion {
	case KubernetesService:
		return r.syncIstio(ctx, instance)
	case KnativeServiceV1Alpha1:
		return r.syncKnative(ctx, instance)
	default:
		instance.Status.MarkHasNotService("UnsupportedAPIVersion", "%s", apiVersion)
		err := r.Status().Update(ctx, instance)
		return reconcile.Result{}, err
	}
}

func (r *ReconcileCanary) finalize(context context.Context, instance *iter8v1alpha1.Canary) (reconcile.Result, error) {
	log := Logger(context)
	log.Info("finalizing")

	apiVersion := instance.Spec.TargetService.APIVersion
	switch apiVersion {
	case KubernetesService:
		return r.finalizeIstio(context, instance)
	}

	return reconcile.Result{}, removeFinalizer(context, r, instance, Finalizer)
}

func addFinalizerIfAbsent(context context.Context, r *ReconcileCanary, instance *iter8v1alpha1.Canary, fName string) (err error) {
	for _, finalizer := range instance.ObjectMeta.GetFinalizers() {
		if finalizer == fName {
			return
		}
	}

	instance.SetFinalizers(append(instance.GetFinalizers(), Finalizer))
	if err = r.Update(context, instance); err != nil {
		Logger(context).Info("setting finalizer failed. (retrying)", "error", err)
	}

	return
}

func removeFinalizer(context context.Context, r *ReconcileCanary, instance *iter8v1alpha1.Canary, fName string) (err error) {
	finalizers := make([]string, 0)
	for _, f := range instance.GetFinalizers() {
		if f != fName {
			finalizers = append(finalizers, f)
		}
	}
	instance.SetFinalizers(finalizers)
	if err = r.Update(context, instance); err != nil {
		Logger(context).Info("setting finalizer failed. (retrying)", "error", err)
	}

	Logger(context).Info("FinalizerRemoved")
	return
}

// Logger gets the logger from the context.
func Logger(ctx context.Context) logr.Logger {
	return ctx.Value(loggerKey).(logr.Logger)
}
