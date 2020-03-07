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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	iter8cache "github.com/iter8-tools/iter8-controller/pkg/controller/experiment/cache"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/routing"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/targets"
	iter8notifier "github.com/iter8-tools/iter8-controller/pkg/notifier"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
)

var log = logf.Log.WithName("experiment-controller")

type loggerKeyType string

const (
	KubernetesService      = "v1"
	KnativeServiceV1Alpha1 = "serving.knative.dev/v1alpha1"

	Iter8Controller = "iter8-controller"
	Finalizer       = "finalizer.iter8-tools"
	loggerKey       = loggerKeyType("logger")
)

// Add creates a new Experiment Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, stop <-chan struct{}) error {
	r, err := newReconciler(mgr, stop)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, stop <-chan struct{}) (*ReconcileExperiment, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "unable to get client config")
		return nil, err
	}

	ic, err := istioclient.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "Failed to create istio client")
		return nil, err
	}

	k8sCache := mgr.GetCache()

	// Set up notifier configmap handler
	nc := iter8notifier.NewNotificationCenter(log)
	err = nc.RegisterHandler(k8sCache)
	if err != nil {
		log.Error(err, "Failed to register notifier config handlers")
		return nil, err
	}

	iter8Cache := iter8cache.New(k8sCache, log)

	return &ReconcileExperiment{
		Client:             mgr.GetClient(),
		istioClient:        ic,
		scheme:             mgr.GetScheme(),
		eventRecorder:      mgr.GetEventRecorderFor(Iter8Controller),
		notificationCenter: nc,
		iter8Cache:         iter8Cache,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileExperiment) error {
	// Create a new controller
	c, err := controller.New("experiment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	deploymentPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			name, namespace := e.Meta.GetName(), e.Meta.GetNamespace()
			key := r.iter8Cache.GetExperimentKey(name, namespace)
			if key == "" {
				return false
			}

			log.Info("TargetDetected", "Experiment", key, "Target", name+"."+namespace)

			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			name, namespace := e.MetaOld.GetName(), e.MetaOld.GetNamespace()
			key := r.iter8Cache.GetExperimentKey(name, namespace)
			if key == "" {
				return false
			}
			if e.ObjectOld != e.ObjectNew {
				r.iter8Cache.AbortExperiment(key)
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			name, namespace := e.Meta.GetName(), e.Meta.GetNamespace()
			key := r.iter8Cache.GetExperimentKey(name, namespace)
			if key == "" {
				return false
			}

			r.iter8Cache.AbortExperiment(key)
			return true
		},
	}

	deploymentToExperiment := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			name, namespace := a.Meta.GetName(), a.Meta.GetNamespace()
			key := r.iter8Cache.GetExperimentKey(name, namespace)
			experimentName, experimentNamespace := resolveExperiemntLabel(key)
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      experimentName,
						Namespace: experimentNamespace,
					},
				},
			}
		},
	)

	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: deploymentToExperiment},
		deploymentPredicate)

	// Watch for changes to Experiment
	err = c.Watch(&source.Kind{Type: &iter8v1alpha1.Experiment{}}, &handler.EnqueueRequestForObject{},
		// Ignore status update event
		predicate.GenerationChangedPredicate{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldInstance, _ := e.ObjectOld.(*iter8v1alpha1.Experiment)
				newInstance, _ := e.ObjectNew.(*iter8v1alpha1.Experiment)
				// Ignore event of revert changes in action field
				if len(oldInstance.Action) > 0 && len(newInstance.Action) == 0 {
					return false
				}

				// Ignore event of metrics load
				if len(oldInstance.Metrics) == 0 && len(newInstance.Metrics) > 0 {
					return false
				}

				return true
			},
		})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileExperiment{}

// ReconcileExperiment reconciles a Experiment object
type ReconcileExperiment struct {
	client.Client
	scheme             *runtime.Scheme
	eventRecorder      record.EventRecorder
	notificationCenter *iter8notifier.NotificationCenter
	istioClient        istioclient.Interface
	iter8Cache         iter8cache.Interface

	targets *targets.Targets
	rules   *routing.IstioRoutingRules
}

// Reconcile reads that state of the cluster for a Experiment object and makes changes based on the state read
// and what is in the Experiment.Spec
// +kubebuilder:rbac:groups=iter8.tools,resources=experiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iter8.tools,resources=experiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=destinationrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=serving.knative.dev,resources=revisions,verbs=get;list;watch
// +kubebuilder:rbac:groups=serving.knative.dev,resources=revisions/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
func (r *ReconcileExperiment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	// Fetch the Experiment instance
	instance := &iter8v1alpha1.Experiment{}
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
	log.Info("reconciling")
	// Add finalizer to the experiment object
	if err = addFinalizerIfAbsent(ctx, r, instance, Finalizer); err != nil {
		return reconcile.Result{}, err
	}

	// Check whether object has been deleted
	if instance.DeletionTimestamp != nil {
		return r.finalize(ctx, instance)
	}

	r.iter8Cache.RegisterExperiment(instance)

	// Stop right here if the experiment is completed.
	completed := instance.Status.GetCondition(iter8v1alpha1.ExperimentConditionExperimentCompleted)
	if completed != nil && completed.Status == corev1.ConditionTrue {
		log.Info("RolloutCompleted", "Use a different name for experiment object to trigger a new experiment", "")
		return reconcile.Result{}, nil
	}

	if pause := r.pauseOrResume(ctx, instance); pause {
		log.Info("ExperimentPaused")
		return reconcile.Result{}, nil
	}

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

	// Update Grafana URL when experiment is created
	if instance.Status.StartTimestamp == "" {
		ts := now.UTC().UnixNano() / int64(time.Millisecond)
		instance.Status.StartTimestamp = strconv.FormatInt(ts, 10)
		updateGrafanaURL(instance, getServiceNamespace(instance))
	}

	instance.Status.InitializeConditions()

	// Sync metric definitions from the config map
	metricsSycned := instance.Status.GetCondition(iter8v1alpha1.ExperimentConditionMetricsSynced)
	if metricsSycned == nil || metricsSycned.Status != corev1.ConditionTrue {
		if err := readMetrics(ctx, r, instance); err != nil && !validUpdateErr(err) {
			r.MarkSyncMetricsError(ctx, instance, "Fail to read metrics: %v", err)

			if err := r.Status().Update(ctx, instance); err != nil && !validUpdateErr(err) {
				log.Info("Fail to update status: %v", err)
				// End experiment
				return reconcile.Result{}, nil
			}

			return reconcile.Result{}, nil
		}
		r.MarkSyncMetrics(ctx, instance)
	}

	apiVersion := instance.Spec.TargetService.APIVersion

	switch apiVersion {
	case KubernetesService:
		return r.syncKubernetes(ctx, instance)
	case KnativeServiceV1Alpha1:
		return r.syncKnative(ctx, instance)
	default:
		instance.Status.MarkTargetsError("UnsupportedAPIVersion", "%s", apiVersion)
		err := r.Status().Update(ctx, instance)
		return reconcile.Result{}, err
	}
}

func (r *ReconcileExperiment) init(context context.Context, instance *iter8v1alpha1.Experiment) {

}

func (r *ReconcileExperiment) finalize(context context.Context, instance *iter8v1alpha1.Experiment) (reconcile.Result, error) {
	log := Logger(context)
	log.Info("finalizing")

	apiVersion := instance.Spec.TargetService.APIVersion
	switch apiVersion {
	case KubernetesService:
		return r.finalizeIstio(context, instance)
	case KnativeServiceV1Alpha1:
		return r.finalizeKnative(context, instance)
	}

	return reconcile.Result{}, removeFinalizer(context, r, instance, Finalizer)
}

func (r *ReconcileExperiment) pause(context context.Context, instance *iter8v1alpha1.Experiment) bool {
	prevPhase := instance.Status.Phase
	switch instance.Action {
	case iter8v1alpha1.ActionPause:
		instance.Status.Phase = iter8v1alpha1.PhasePause
		log.Info("UserAction", "pause experiment", "")
	case iter8v1alpha1.ActionResume:
		instance.Status.Phase = iter8v1alpha1.PhaseProgressing
		log.Info("UserAction", "resume experiment", "")
	default:
		return false
	}

	if prevPhase != instance.Status.Phase {
		if err := r.Status().Update(context, instance); err != nil && !validUpdateErr(err) {
			log.Info("Fail to update status: %v", err)
		}
	}

	// clear instance action
	instance.Action = ""
	if err := r.Update(context, instance); err != nil && !validUpdateErr(err) {
		log.Error(err, "Fail to revert action", "")
	}

	return instance.Status.Phase == iter8v1alpha1.PhasePause
}

func (r *ReconcileExperiment) addLabelToDeployment(ctx context.Context, name, namespace, label, val string) error {
	d := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, d)
	if err != nil {
		return err
	}

	d.SetLabels(map[string]string{label: val})

	return r.Update(ctx, d)
}
