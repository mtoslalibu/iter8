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
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.ibm.com/istio-research/iter8-controller/pkg/analytics/checkandincrement"
	"github.ibm.com/istio-research/iter8-controller/pkg/apis/iter8/v1alpha1"
)

// Flags holds the command line flags or defaults for settings in the user's environment.
// See EnvironmentFlags for a list of supported fields.
var Flags = initializeFlags()
var compoptions = cmpopts.IgnoreTypes(metav1.TypeMeta{}, metav1.ObjectMeta{}, metav1.Time{})

// EnvironmentFlags define the flags that are needed to run the e2e tests.
type EnvironmentFlags struct {
	Namespace string // K8s namespace (blank by default, to be overwritten by test suite)
}

func initializeFlags() *EnvironmentFlags {
	var f EnvironmentFlags

	flag.StringVar(&f.Namespace, "namespace", "",
		"Provide the namespace you would like to use for these tests.")

	return &f
}

// GetClient returns generic Kube client configure with experiment scheme
func GetClient() client.Client {
	// Get a config to talk to the apiserver

	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Errorf("unable to set up client config (%v)", err))
	}

	sch := scheme.Scheme
	if err := v1alpha1.AddToScheme(sch); err != nil {
		panic(fmt.Errorf("unable to add scheme (%v)", err))
	}

	cl, err := client.New(cfg, client.Options{Scheme: sch})
	if err != nil {
		panic(fmt.Errorf("unable to set up client config (%v)", err))
	}

	return cl
}

type testCase struct {
	mocks []mocks

	// Object to reconcile
	object runtime.Object

	wantResults []runtime.Object
}

type mocks struct {
	request  checkandincrement.Request
	response checkandincrement.Response
}

func (tc *testCase) createObject(ctx context.Context, cl client.Client) error {
	return cl.Create(ctx, tc.object)
}

func (tc *testCase) checkHasResults(ctx context.Context, cl client.Client) error {
	for _, result := range tc.wantResults {
		retries := 5
		for {
			accessor, err := meta.Accessor(result)
			if err != nil {
				return err
			}

			obj, err := scheme.Scheme.New(result.GetObjectKind().GroupVersionKind())
			if err != nil {
				return err
			}

			err = cl.Get(ctx, client.ObjectKey{Namespace: accessor.GetNamespace(), Name: accessor.GetName()}, obj)
			if err != nil {
				return err
			}

			if diff := cmp.Diff(result, obj, compoptions); diff != "" {
				if retries == 0 {
					return fmt.Errorf("unexpected reponse diff (-want, +got) = %v", diff)
				}
			} else {
				break
			}

			time.Sleep(time.Second)

			retries--
		}
	}
	return nil
}
