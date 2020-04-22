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
package e2e

import (
	"testing"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	analtyicsapi "github.com/iter8-tools/iter8-controller/pkg/analytics/api"
	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/routing"
	"github.com/iter8-tools/iter8-controller/test"
)

const (
	//Bookinfo sample constant values
	ReviewsV1Image = "istio/examples-bookinfo-reviews-v1:1.11.0"
	ReviewsV2Image = "istio/examples-bookinfo-reviews-v2:1.11.0"
	ReviewsV3Image = "istio/examples-bookinfo-reviews-v3:1.11.0"
	RatingsImage   = "istio/examples-bookinfo-ratings-v1:1.11.0"

	ReviewsPort = 9080
	RatingsPort = 9080
)

// TestKubernetesExperiment tests various experiment scenarios on Kubernetes platform
func TestKubernetesExperiment(t *testing.T) {
	service := test.StartAnalytics()
	defer service.Close()
	testCases := map[string]testCase{
		"rollforward": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getFastKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentSuccess,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("rollforward"),
		"rollbackward": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetFailureMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getFastKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentFailure,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
			}
		}("rollbackward"),
		"ongoingdelete": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object:    getSlowKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.CheckServiceFound,
				wantResults: []runtime.Object{
					// rollback to baseline
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
				postHook: test.DeleteExperiment(name, Flags.Namespace),
			}
		}("ongoingdelete"),
		"completedelete": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object:    getFastKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.CheckExperimentFinished,
				frozenObjects: []runtime.Object{
					// desired end-of-experiment status
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
				postHook: test.DeleteExperiment(name, Flags.Namespace),
			}
		}("completedelete"),
		"abortexperiment": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetAbortExperimentResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getSlowKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentFailure,
				),
				wantResults: []runtime.Object{
					// rollback to baseline
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
			}
		}("abortexperiment"),
		"emptycriterion": func(name string) testCase {
			return testCase{
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getDefaultKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2"),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentSuccess,
				),
				wantResults: []runtime.Object{
					// rollforward
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("emptycriterion"),
		"externalreference": func(name string) testCase {
			return testCase{
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getSampleEdgeVirtualService(),
				},
				preHook: []test.Hook{
					test.DeleteObjectIfExists(getStableDestinationRule("reviews", "", getReviewsDeployment("v2"))),
					test.DeleteObjectIfExists(getStableVirtualService("reviews", "")),
				},
				object:    getExperimentWithExternalReference(name, "reviews", "reviews-v1", "reviews-v2"),
				wantState: test.CheckExperimentFinished,
				wantResults: []runtime.Object{
					// rollforward
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getSampleStableEdgeVirtualService(),
				},
				finalizers: []test.Hook{
					test.DeleteObject(getStableDestinationRule("reviews", name, getReviewsDeployment("v2"))),
					test.DeleteObject(getSampleStableEdgeVirtualService()),
				},
			}
		}("externalreference"),
		"cleanupdelete": func(name string) testCase {
			return testCase{
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getCleanUpDeleteExperiment(name, "reviews", "reviews-v1", "reviews-v2"),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentSuccess,
				),
				postHook: test.CheckObjectDeleted(getReviewsDeployment("v1")),
			}
		}("cleanupdelete"),
		"greedy-rollforward": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getGreedyFastKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentSuccess,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("greedy-rollforward"),
		"greedy-rollbackward": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetFailureMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getGreedyFastKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentFailure,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
			}
		}("greedy-rollbackward"),
		"reward-rollforward": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getRewardFastKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentFinished,
					test.CheckExperimentSuccess,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("reward-rollforward"),
		"pauseresume": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object: getPauseExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.WantAllStates(
					test.CheckExperimentPause,
				),
				postHook: test.ResumeExperiment(getPauseExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL())),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("pauseresume"),
		"deletebaseline": func(name string) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetSuccessMockResponse(),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
				},
				object:    getSlowKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
				wantState: test.CheckServiceFound,
				postHook:  test.DeleteObject(getReviewsDeployment("v1")),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("deletebaseline"),
	}

	runTestCases(t, service, testCases)
}

func getReviewsService() runtime.Object {
	return test.NewKubernetesService("reviews", Flags.Namespace).
		WithSelector(map[string]string{"app": "reviews"}).
		WithPorts(map[string]int{"http": ReviewsPort}).
		Build()
}

func getRatingsService() runtime.Object {
	return test.NewKubernetesService("ratings", Flags.Namespace).
		WithSelector(map[string]string{"app": "ratings"}).
		WithPorts(map[string]int{"http": RatingsPort}).
		Build()
}

func getReviewsDeployment(version string) runtime.Object {
	labels := map[string]string{
		"app":     "reviews",
		"version": version,
	}

	image := ""

	switch version {
	case "v1":
		image = ReviewsV1Image
	case "v2":
		image = ReviewsV2Image
	case "v3":
		image = ReviewsV3Image
	default:
		image = ReviewsV1Image
	}

	return test.NewKubernetesDeployment("reviews-"+version, Flags.Namespace).
		WithLabels(labels).
		WithContainer("reviews", image, ReviewsPort).
		Build()
}

func getPauseExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	exp := getFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost)
	exp.Action = iter8v1alpha1.ActionPause
	return exp
}

func getCleanUpDeleteExperiment(name, serviceName, baseline, candidate string) *iter8v1alpha1.Experiment {
	exp := getDefaultKubernetesExperiment(name, serviceName, baseline, candidate)
	exp.Spec.CleanUp = iter8v1alpha1.CleanUpDelete
	return exp
}

func getDefaultKubernetesExperiment(name, serviceName, baseline, candidate string) *iter8v1alpha1.Experiment {
	exp := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidate).
		Build()

	onesec := "1s"
	one := 1
	exp.Spec.TrafficControl.Interval = &onesec
	exp.Spec.TrafficControl.MaxIterations = &one

	return exp
}

func getExperimentWithExternalReference(name, serviceName, baseline, candidate string) *iter8v1alpha1.Experiment {
	exp := getDefaultKubernetesExperiment(name, serviceName, baseline, candidate)
	exp.Spec.RoutingReference = &corev1.ObjectReference{
		Name:       "reviews-external",
		APIVersion: v1alpha3.SchemeGroupVersion.String(),
		Kind:       "VirtualService",
	}

	return exp
}

func getFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidate).
		WithAnalyticsHost(analyticsHost).
		WithDummySuccessCriterion().
		Build()

	onesec := "1s"
	one := 1
	experiment.Spec.TrafficControl.Interval = &onesec
	experiment.Spec.TrafficControl.MaxIterations = &one

	return experiment
}

func getGreedyFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	experiment := getFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost)

	greedy := "epsilon_greedy"
	experiment.Spec.TrafficControl.Strategy = &greedy

	return experiment
}

func getRewardFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	experiment := getFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost)

	experiment.Spec.Analysis.Reward = &iter8v1alpha1.Reward{
		MetricName: "iter8_latency",
		MinMax: &iter8v1alpha1.MinMax{
			Min: 0,
			Max: 10,
		},
	}

	return experiment
}

func getSlowKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidate).
		WithAnalyticsHost(analyticsHost).
		WithDummySuccessCriterion().
		Build()

	tensecs := "10s"
	two := 2
	experiment.Spec.TrafficControl.Interval = &tensecs
	experiment.Spec.TrafficControl.MaxIterations = &two

	return experiment
}

func getStableDestinationRule(serviceName, name string, obj runtime.Object) runtime.Object {
	deploy := obj.(*appsv1.Deployment)

	return routing.NewDestinationRule(serviceName, name, Flags.Namespace).
		WithStableDeployment(deploy).
		Build()
}

func getStableVirtualService(serviceName, name string) runtime.Object {
	return routing.NewVirtualService(serviceName, name, Flags.Namespace).
		WithNewStableSet(serviceName).
		Build()
}

func getSampleEdgeVirtualService() runtime.Object {
	return &v1alpha3.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reviews-external",
			Namespace: Flags.Namespace,
		},
		Spec: networkingv1alpha3.VirtualService{
			Hosts:    []string{"reviews.com"},
			Gateways: []string{"reviews-gateway"},
			Http: []*networkingv1alpha3.HTTPRoute{
				{
					Route: []*networkingv1alpha3.HTTPRouteDestination{
						{
							Destination: &networkingv1alpha3.Destination{
								Host: "reviews",
								Port: &networkingv1alpha3.PortSelector{
									Number: 9080,
								},
							},
						},
					},
				},
			},
		},
	}
}

func getSampleStableEdgeVirtualService() runtime.Object {
	vs := getSampleEdgeVirtualService()
	vs.(*v1alpha3.VirtualService).Spec.Http[0].Route[0].Destination.Subset = "stable"
	vs.(*v1alpha3.VirtualService).Spec.Http[0].Route[0].Weight = 100

	return vs
}
