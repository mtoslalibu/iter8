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
	"sync"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	iter8v1alpha1 "github.com/iter8-tools/iter8-controller/pkg/apis/iter8/v1alpha1"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/cache/targets"
	_ "github.com/iter8-tools/iter8-controller/pkg/controller/experiment/routing"
	"github.com/iter8-tools/iter8-controller/pkg/controller/experiment/util"
)

type Interface interface {
	// Given name and namespace of the target deployment, return the experiment key
	DeploymentToExperiment(name, namespace string) (experiment, experimentNamespace string, exist bool)
	// Given name and namespace of the target service, return the experiment key
	ServiceToExperiment(name, namespace string) (experiment, experimentNamespace string, exist bool)
	RegisterExperiment(instance *iter8v1alpha1.Experiment)
	RemoveExperiment(instance *iter8v1alpha1.Experiment)
	Print()
}

var _ Interface = &Impl{}

// ExperimentAbstract includes basic info for one Experiment
type ExperimentAbstract struct {
	namespace       string
	phase           iter8v1alpha1.Phase
	targetsAbstract *targets.Abstract
}

// Impl is the implementation of Iter8Cache
type Impl struct {
	k8sCache cache.Cache
	// an ExperimentAbstract store with experimentName.experimentNamespace as key for access
	experimentAbstractStore map[string]*ExperimentAbstract

	// a lookup map from target to experiment
	// targetName.targetNamespace -> experimentName.experimentNamespace
	deployment2Experiment map[string]string

	// a lookup map from target service to experiment
	service2Experiment map[string]string

	logger logr.Logger

	m sync.RWMutex
}

func New(k8sCache cache.Cache, logger logr.Logger) Interface {
	return &Impl{
		k8sCache:                k8sCache,
		experimentAbstractStore: make(map[string]*ExperimentAbstract),
		deployment2Experiment:   make(map[string]string),
		service2Experiment:      make(map[string]string),
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
		namespace:       instance.Namespace,
		targetsAbstract: targets.NewAbstract(instance),
	}
	c.experimentAbstractStore[eakey] = ea
	service := instance.Spec.TargetService.Name
	baseline := instance.Spec.TargetService.Baseline
	candidate := instance.Spec.TargetService.Candidate
	c.service2Experiment[targetKey(service, targetNamespace)] = eakey
	c.deployment2Experiment[targetKey(baseline, targetNamespace)] = eakey
	c.deployment2Experiment[targetKey(candidate, targetNamespace)] = eakey
}

// DeploymentToExperiment returns the experiment key given name and namespace of target deployment
func (c *Impl) DeploymentToExperiment(targetName, targetNamespace string) (string, string, bool) {
	tKey := targetKey(targetName, targetNamespace)
	if _, ok := c.deployment2Experiment[tKey]; !ok {
		return "", "", false
	}
	name, namespace := resolveExperimentKey(c.deployment2Experiment[tKey])

	return name, namespace, true
}

// ServiceToExperiment returns the experiment key given name and namespace of target service
func (c *Impl) ServiceToExperiment(targetName, targetNamespace string) (string, string, bool) {
	tKey := targetKey(targetName, targetNamespace)
	if _, ok := c.service2Experiment[tKey]; !ok {
		return "", "", false
	}

	name, namespace := resolveExperimentKey(c.service2Experiment[tKey])

	return name, namespace, true
}

func (c *Impl) Print() {
	c.logger.Info("CacheStatus", "deployment2experiment", c.deployment2Experiment,
		"service2experiment", c.service2Experiment)
}

func (c *Impl) RemoveExperiment(instance *iter8v1alpha1.Experiment) {
	ea, ok := c.experimentAbstractStore[experimentKey(instance)]
	if !ok {
		return
	}

	ta := ea.targetsAbstract
	targetNamespace := ta.Namespace
	delete(c.service2Experiment, targetKey(ta.ServiceName, targetNamespace))
	for name := range ta.Status {
		delete(c.deployment2Experiment, targetKey(name, targetNamespace))
	}
}
