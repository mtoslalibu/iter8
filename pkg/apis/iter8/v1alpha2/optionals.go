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

package v1alpha2

import (
	"time"
)

const (
	// DefaultRewardMetric indicate whether a metric is a reward by default, which is false
	DefaultRewardMetric bool = false

	// DefaultZeroToOne indicate whether the value range of metric is from 0 to 1  by default, which is false
	DefaultZeroToOne bool = false

	// DefaultStrategy is the default value for strategy, which is progressive
	DefaultStrategy StrategyType = StrategyProgressive

	// DefaultOnTermination is the default value for onTermination, which is to_winner
	DefaultOnTermination OnTerminationType = OnTerminationToWinner

	// DefaultPercentage is the default traffic percentage used in experiment, which is 100
	DefaultPercentage int32 = 100

	// DefaultMaxIncrement is the default maxIncrement for traffic update, which is 2
	DefaultMaxIncrement int32 = 2

	// DefaultDuration is the default duration for an interval, which is 30 seconds
	DefaultDuration time.Duration = time.Second * 30

	// DefaultMaxIterations is the default number of iterations, which is 100
	DefaultMaxIterations int32 = 100
)

// ServiceNamespace gets the namespace for targets
func (e *Experiment) ServiceNamespace() string {
	serviceNamespace := e.Spec.Service.Namespace
	if serviceNamespace == "" {
		serviceNamespace = e.Namespace
	}
	return serviceNamespace
}

// HasRewardMetric indicates whether this criterion uses a reward metric or not
func (c *Criterion) HasRewardMetric() bool {
	if c.IsReward == nil {
		return DefaultRewardMetric
	}
	return *c.IsReward
}

// CutOffOnViolation indicates whether traffic should be cutoff to a target if threshold is violated
func (t *Threshold) CutOffOnViolation() bool {
	if t.CutoffTrafficOnViolation == nil {
		return false
	}
	return *t.CutoffTrafficOnViolation
}

// GetStrategy gets the specified(or default) strategy used for traffic control
func (t *TrafficControl) GetStrategy() StrategyType {
	if t.Strategy == nil {
		return DefaultStrategy
	}
	return *t.Strategy
}

// GetOnTermination returns specified(or default) onTermination strategy for traffic controller
func (t *TrafficControl) GetOnTermination() OnTerminationType {
	if t.OnTermination == nil {
		return DefaultOnTermination
	}
	return *t.OnTermination
}

// GetPercentage returns specified(or default) experiment traffic percentage
func (t *TrafficControl) GetPercentage() int32 {
	if t.Percentage == nil {
		return DefaultPercentage
	}
	return *t.Percentage
}

// GetMaxIncrements returns specified(or default) maxIncrements for each traffic update
func (t *TrafficControl) GetMaxIncrements() int32 {
	if t.MaxIncrement == nil {
		return DefaultMaxIncrement
	}
	return *t.MaxIncrement
}

// GetInterval returns specified(or default) interval for each duration
func (d *Duration) GetInterval() (time.Duration, error) {
	if d.Interval == nil {
		return DefaultDuration, nil
	}
	return time.ParseDuration(*d.Interval)
}

// GetMaxIterations returns specified(or default) max of iterations
func (d *Duration) GetMaxIterations() int32 {
	if d.MaxIterations == nil {
		return DefaultMaxIterations
	}
	return *d.MaxIterations
}

// IsZeroToOne returns specified(or default) zeroToOne value
func (r *RatioMetric) IsZeroToOne() bool {
	if r.ZeroToOne == nil {
		return DefaultZeroToOne
	}
	return *r.ZeroToOne
}
