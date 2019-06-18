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

	"k8s.io/apimachinery/pkg/runtime"

	cai "github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	iter8v1alpha1 "github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.ibm.com/istio-research/iter8-controller/pkg/controller/experiment"
	"github.ibm.com/istio-research/iter8-controller/test"
	appsv1 "k8s.io/api/apps/v1"
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
		"rollforward": testCase{
			mocks: map[string]cai.Response{
				"rollforward": test.GetSuccessMockResponse(),
			},
			initObjects: []runtime.Object{
				getReviewsService(),
				getRatingsService(),
				getReviewsDeployment("v1"),
				getReviewsDeployment("v2"),
				getRatingsDeployment(),
			},
			object:    getFastKubernetesExperiment("rollforward", "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
			wantState: test.CheckExperimentFinished,
			wantResults: []runtime.Object{
				getStableDestinationRule("reviews", "rollforward", getReviewsDeployment("v2")),
				getStableVirtualService("reviews", "rollforward"),
			},
		},
		"rollbackward": testCase{
			mocks: map[string]cai.Response{
				"rollbackward": test.GetFailureMockResponse(),
			},
			initObjects: []runtime.Object{
				getReviewsService(),
				getRatingsService(),
				getReviewsDeployment("v1"),
				getReviewsDeployment("v2"),
				getRatingsDeployment(),
			},
			object:    getFastKubernetesExperiment("rollbackward", "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
			wantState: test.CheckExperimentFinished,
			wantResults: []runtime.Object{
				getStableDestinationRule("reviews", "rollbackward", getReviewsDeployment("v1")),
				getStableVirtualService("reviews", "rollbackward"),
			},
		},
		// "completeDelete": testCase{
		// 	mocks: map[string]cai.Response{
		// 		"completeDelete": test.GetSuccessMockResponse(),
		// 	},
		// 	initObjects: []runtime.Object{
		// 		getReviewsService(),
		// 		getRatingsService(),
		// 		getReviewsDeployment("v1"),
		// 		getReviewsDeployment("v2"),
		// 		getRatingsDeployment(),
		// 	},
		// 	object:    getSlowKubernetesExperiment("completeDelete", "reviews", "reviews-v1", "reviews-v2", service.GetURL()),
		// 	wantState: test.CheckExperimentFinished,
		// 	frozenObjects: []runtime.Object{
		// 		getStableDestinationRule("reviews", "completeDelete", getReviewsDeployment("v2")),
		// 		getStableVirtualService("reviews", "completeDelete"),
		// 	},
		// 	postHook: test.DeleteExperiment("completeDelete", Flags.Namespace),
		// },
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
		WithTemplateLabels(labels).
		WithContainer("reviews", image, ReviewsPort).
		Build()
}

func getRatingsDeployment() runtime.Object {
	labels := map[string]string{
		"app":     "ratings",
		"version": "v1",
	}

	return test.NewKubernetesDeployment("ratings-v1", Flags.Namespace).
		WithTemplateLabels(labels).
		WithContainer("ratings", RatingsImage, RatingsPort).
		Build()
}

func getDefaultKubernetesExperiment(name, serviceName, baseline, candidate string) *iter8v1alpha1.Experiment {
	return test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidate).
		Build()
}

func getFastKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidate).
		WithAnalyticsHost(analyticsHost).
		Build()

	onesec := "1s"
	one := 1
	experiment.Spec.TrafficControl.Interval = &onesec
	experiment.Spec.TrafficControl.MaxIterations = &one

	return experiment
}

func getSlowKubernetesExperiment(name, serviceName, baseline, candidate, analyticsHost string) *iter8v1alpha1.Experiment {
	experiment := test.NewExperiment(name, Flags.Namespace).
		WithKubernetesTargetService(serviceName, baseline, candidate).
		WithAnalyticsHost(analyticsHost).
		Build()

	twentysecs := "20s"
	one := 1
	experiment.Spec.TrafficControl.Interval = &twentysecs
	experiment.Spec.TrafficControl.MaxIterations = &one

	return experiment
}

func getStableDestinationRule(serviceName, name string, obj runtime.Object) runtime.Object {
	deploy := obj.(*appsv1.Deployment)

	return experiment.NewDestinationRule(serviceName, name, Flags.Namespace).
		WithStableDeployment(deploy).
		Build()
}

func getStableVirtualService(serviceName, name string) runtime.Object {
	return experiment.NewVirtualService(serviceName, name, Flags.Namespace).
		WithStableSet(serviceName).
		Build()
}
