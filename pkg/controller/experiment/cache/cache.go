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

package cache

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sync"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/cache/targets"
	_ "github.com/iter8-tools/iter8-controller/pkg/controller/experiment/routing"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

type Interface interface {
	// Given name and namespace of the target, return the experiment key
	GetExperimentKey(name, namespace string) (experimentKey string)
	RegisterExperiment(instance *iter8v1alpha1.Experiment)
	AbortExperiment(experimentKey string)
}

var _ Interface = &Impl{}

// ExperimentAbstract includes basic info for one Experiment
type ExperimentAbstract struct {
	namespace       string
	phase           iter8v1alpha1.Phase
	targetsAbstract *targets.TargetsAbstract
}

// Impl is the implementation of Iter8Cache
type Impl struct {
	k8sCache cache.Cache
	// an ExperimentAbstract store with experimentName.experimentNamespace as key for access
	experimentAbstractStore map[string]*ExperimentAbstract

	// a namespace-scoped lookup map from target to experiment
	// targetName.targetNamespace -> experimentName.experimentNamespace
	target2Experiment map[string]string

	logger logr.Logger

	m sync.RWMutex
}

func New(k8sCache cache.Cache, logger logr.Logger) Interface {
	return &Impl{
		k8sCache:                k8sCache,
		experimentAbstractStore: make(map[string]*ExperimentAbstract),
		target2Experiment:       make(map[string]string),
		logger:                  logger,
	}
}

func (c *Impl) RegisterExperiment(instance *iter8v1alpha1.Experiment) {
	eakey := experimentKey(instance)
	if _, ok := c.experimentAbstractStore[eakey]; ok {
		return
	}
	targetNamespace := util.GetServiceNamespace(instance)

	ea := &ExperimentAbstract{
		namespace: instance.Namespace,
		targetsAbstract: &targets.TargetsAbstract{
			Namespace: targetNamespace,
		},
	}
	c.experimentAbstractStore[eakey] = ea
	baseline := instance.Spec.TargetService.Baseline
	candidate := instance.Spec.TargetService.Baseline
	c.target2Experiment[targetKey(baseline, targetNamespace)] = eakey
	c.target2Experiment[targetKey(candidate, targetNamespace)] = eakey
}

func experimentKey(instance *iter8v1alpha1.Experiment) string {
	return instance.Name + "." + instance.Namespace
}

func targetKey(name, namespace string) string {
	return name + "." + namespace
}

// GetExperimentKey returns the experiment key given name and namespace of target
func (c *Impl) GetExperimentKey(targetName, targetNamespace string) (experimentKey string) {
	tKey := targetKey(targetName, targetNamespace)
	if _, ok := c.target2Experiment[tKey]; !ok {
		return ""
	}

	return c.target2Experiment[tKey]
}

func (c *Impl) AbortExperiment(experimentKey string) {
	ea, ok := c.experimentAbstractStore[experimentKey]
	if !ok {
		return
	}

	ea.phase = iter8v1alpha1.PhaseCompleted
}
