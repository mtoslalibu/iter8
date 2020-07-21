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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	analtyicsapi "github.com/iter8-tools/iter8-controller/pkg/analytics/api/v1alpha2"
	iter8v1alpha2 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha2"
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
func TestExperiment(t *testing.T) {
	service := test.StartAnalytics()
	defer service.Close()
	testCases := map[string]testCase{
		"rolltowinner": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetRollToWinnerMockResponse(exp, 0),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object: exp,
				wantState: test.WantAllStates(
					test.CheckExperimentCompleted,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
			}
		}("rolltowinner", getFastKubernetesExperiment("rolltowinner", "reviews", "reviews-v1", service.GetURL(), []string{"reviews-v2", "reviews-v3"})),
		"rollbackward": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetRollbackMockResponse(exp),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object: exp,
				wantState: test.WantAllStates(
					test.CheckExperimentCompleted,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
			}
		}("rollbackward", getFastKubernetesExperiment("rollbackward", "reviews", "reviews-v1", service.GetURL(), []string{"reviews-v2", "reviews-v3"})),
		"ongoingdelete": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetRollToWinnerMockResponse(exp, 0),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object:    exp,
				wantState: test.CheckServiceFound,
				wantResults: []runtime.Object{
					// rollback to baseline
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
				postHook: test.DeleteExperiment("ongoingdelete", Flags.Namespace),
			}
		}("ongoingdelete", getSlowKubernetesExperiment("ongoingdelete", "reviews", "reviews-v1", service.GetURL(), []string{"reviews-v2", "reviews-v3"})),
		"completedelete": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetRollToWinnerMockResponse(exp, 0),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object:    exp,
				wantState: test.CheckExperimentCompleted,
				frozenObjects: []runtime.Object{
					// desired end-of-experiment status
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualService("reviews", name),
				},
				postHook: test.DeleteExperiment(name, Flags.Namespace),
			}
		}("completedelete", getFastKubernetesExperiment("completedelete", "reviews", "reviews-v1", service.GetURL(), []string{"reviews-v2", "reviews-v3"})),
		"abortexperiment": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetAbortExperimentResponse(exp),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object: exp,
				wantState: test.WantAllStates(
					test.CheckExperimentCompleted,
				),
				wantResults: []runtime.Object{
					// rollback to baseline
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
			}
		}("abortexperiment", getSlowKubernetesExperiment("abortexperiment", "reviews", "reviews-v1", service.GetURL(), []string{"reviews-v2", "reviews-v3"})),
		"emptycriterion": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object: exp,
				wantState: test.WantAllStates(
					test.CheckExperimentCompleted,
				),
				wantResults: []runtime.Object{
					// rollforward
					getStableDestinationRule("reviews", name, getReviewsDeployment("v1")),
					getStableVirtualService("reviews", name),
				},
			}
		}("emptycriterion", getDefaultKubernetesExperiment("emptycriterion", "reviews", "reviews-v1", []string{"reviews-v2", "reviews-v3"})),
		"cleanupdelete": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object: exp,
				wantState: test.WantAllStates(
					test.CheckExperimentCompleted,
				),
				postHook: test.CheckObjectDeleted(getReviewsDeployment("v2"), getReviewsDeployment("v3")),
			}
		}("cleanupdelete", getCleanUpDeleteExperiment("cleanupdelete", "reviews", "reviews-v1", []string{"reviews-v2", "reviews-v3"})),
		"attach-gateway": func(name string, exp *iter8v1alpha2.Experiment) testCase {
			return testCase{
				mocks: map[string]analtyicsapi.Response{
					name: test.GetRollToWinnerMockResponse(exp, 0),
				},
				initObjects: []runtime.Object{
					getReviewsService(),
					getReviewsDeployment("v1"),
					getReviewsDeployment("v2"),
					getReviewsDeployment("v3"),
				},
				object: exp,
				wantState: test.WantAllStates(
					test.CheckExperimentCompleted,
				),
				wantResults: []runtime.Object{
					getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
					getStableVirtualServiceWithGateway("reviews", name, "reviews.com", "gateway-testing"),
				},
			}
		}("attach-gateway", getExperimentWithGateway("attach-gateway", "reviews", "reviews-v1", service.GetURL(),
			[]string{"reviews-v2", "reviews-v3"}, "reviews.com", "gateway-testing")),
		// "pauseresume": func(name string, exp *iter8v1alpha2.Experiment) testCase {
		// 	return testCase{
		// 		mocks: map[string]analtyicsapi.Response{
		// 			name: test.GetRollToWinnerMockResponse(exp, 0),
		// 		},
		// 		initObjects: []runtime.Object{
		// 			getReviewsService(),
		// 			getReviewsDeployment("v1"),
		// 			getReviewsDeployment("v2"),
		// 			getReviewsDeployment("v3"),
		// 		},
		// 		object: exp,
		// 		wantState: test.WantAllStates(
		// 			test.CheckExperimentPause,
		// 		),
		// 		postHook: test.ResumeExperiment(exp),
		// 		wantResults: []runtime.Object{
		// 			getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
		// 			getStableVirtualService("reviews", name),
		// 		},
		// 	}
		// }("pauseresume", getPauseExperiment("pauseresume", "reviews", "reviews-v1", service.GetURL(), []string{"reviews-v2", "reviews-v3"})),
		// "deletebaseline": func(name string) testCase {
		// 	return testCase{
		// 		mocks: map[string]analtyicsapi.Response{
		// 			name: test.GetSuccessMockResponse(),
		// 		},
		// 		initObjects: []runtime.Object{
		// 			getReviewsService(),
		// 			getReviewsDeployment("v1"),
		// 			getReviewsDeployment("v2"),
		// 		},
		// 		object:    getSlowKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
		// 		wantState: test.CheckServiceFound,
		// 		postHook:  test.DeleteObject(getReviewsDeployment("v1")),
		// 		wantResults: []runtime.Object{
		// 			getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
		// 			getStableVirtualService("reviews", name),
		// 		},
		// 	}
		// }("deletebaseline"),
		// "duplicate-service": func(name string) testCase {
		// 	return testCase{
		// 		mocks: map[string]analtyicsapi.Response{
		// 			name: test.GetSuccessMockResponse(),
		// 		},
		// 		initObjects: []runtime.Object{
		// 			getReviewsService(),
		// 			getReviewsDeployment("v1"),
		// 			getReviewsDeployment("v2"),
		// 		},
		// 		preHook:   []test.Hook{test.CreateObject(getDefaultKubernetesExperiment(name, "reviews", "reviews-v1", "reviews-v2"))},
		// 		object:    getDefaultKubernetesExperiment(name+"duplicate", "reviews", "reviews-v1", "reviews-v2"),
		// 		wantState: test.CheckServiceNotFound("TargetsNotFound"),
		// 		wantResults: []runtime.Object{
		// 			getStableDestinationRule("reviews", name, getReviewsDeployment("v2")),
		// 			getStableVirtualService("reviews", name),
		// 		},
		// 		finalizers: []test.Hook{
		// 			test.DeleteObject(getDefaultKubernetesExperiment(name+"duplicate", "reviews", "reviews-v1", "reviews-v2")),
		// 		},
		// 	}
		// }("duplicate-service"),
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

func getPauseExperiment(name, serviceName, baseline, analyticsHost string, candidates []string) *iter8v1alpha2.Experiment {
	exp := getFastKubernetesExperiment(name, serviceName, baseline, analyticsHost, candidates)
	exp.Spec.ManualOverride = &iter8v1alpha2.ManualOverride{
		Action: iter8v1alpha2.ActionPause,
	}
	return exp
}

func getCleanUpDeleteExperiment(name, serviceName, baseline string, candidates []string) *iter8v1alpha2.Experiment {
	exp := getDefaultKubernetesExperiment(name, serviceName, baseline, candidates)
	cleanup := true
	exp.Spec.Cleanup = &cleanup
	return exp
}

func getDefaultKubernetesExperiment(name, serviceName, baseline string, candidates []string) *iter8v1alpha2.Experiment {
	exp := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidates).
		Build()

	onesec := "1s"
	one := int32(1)
	exp.Spec.Duration = &iter8v1alpha2.Duration{
		Interval:      &onesec,
		MaxIterations: &one,
	}

	return exp
}

func getFastKubernetesExperiment(name, serviceName, baseline, analyticsHost string, candidates []string) *iter8v1alpha2.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidates).
		WithAnalyticsEndpoint(analyticsHost).
		WithDummyCriterion().
		Build()

	onesec := "1s"
	one := int32(1)
	experiment.Spec.Duration = &iter8v1alpha2.Duration{
		Interval:      &onesec,
		MaxIterations: &one,
	}

	return experiment
}

func getExperimentWithGateway(name, serviceName, baseline, analyticsHost string, candidates []string, host, gw string) *iter8v1alpha2.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidates).
		WithHostInTargetService(host, gw).
		WithAnalyticsEndpoint(analyticsHost).
		WithDummyCriterion().
		Build()

	onesec := "1s"
	one := int32(1)
	experiment.Spec.Duration = &iter8v1alpha2.Duration{
		Interval:      &onesec,
		MaxIterations: &one,
	}

	return experiment
}

func getSlowKubernetesExperiment(name, serviceName, baseline, analyticsHost string, candidates []string) *iter8v1alpha2.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidates).
		WithAnalyticsEndpoint(analyticsHost).
		WithDummyCriterion().
		Build()

	tensecs := "10s"
	two := int32(2)
	experiment.Spec.Duration = &iter8v1alpha2.Duration{
		Interval:      &tensecs,
		MaxIterations: &two,
	}

	return experiment
}

func getStableDestinationRule(serviceName, name string, deploy runtime.Object) runtime.Object {
	d := deploy.(*appsv1.Deployment)
	subset := "dummy"
	return routing.NewDestinationRule(serviceName, name, Flags.Namespace).
		WithSubset(d, subset, 0).
		ProgressingToStable(map[string]string{subset: routing.SubsetStable}).
		WithStableLabel().
		Build()
}

func getStableVirtualService(serviceName, name string) runtime.Object {
	return routing.NewVirtualService(serviceName, name, Flags.Namespace).
		WithMeshGateway().
		ProgressingToStable(map[string]int32{routing.SubsetStable: 100}, serviceName, Flags.Namespace).
		WithStableLabel().
		Build()
}

func getStableVirtualServiceWithGateway(serviceName, name, host, gw string) runtime.Object {
	return routing.NewVirtualService(serviceName, name, Flags.Namespace).
		WithHosts([]string{host}).
		InitGateways().
		WithMeshGateway().
		WithGateways([]string{gw}).
		ProgressingToStable(map[string]int32{routing.SubsetStable: 100}, serviceName, Flags.Namespace).
		WithStableLabel().
		Build()
}
