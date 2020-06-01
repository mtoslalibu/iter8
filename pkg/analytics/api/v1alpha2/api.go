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

// Response from analytics
type Response struct {
	// Timestamp when assessment is made
	Timestamp int64 `json:"timestamp"`

	// Assessment for baseline
	BaselineAssessment VersionAssessment `json:"baseline_assessment"`

	// Assessment for candidates
	CandidateAssessments []CandidateAssessment `json:"candidate_assessments"`

	// TrafficSplitRecommendation contains traffic split recommendations for versions from each algorithm
	// The map is interprted as {name_of_algorithm: {name_of_version: traffic_amount}}
	TrafficSplitRecommendation map[string]map[string]int32 `json:"traffic_split_recommendation"`

	// Assessment for winner
	WinnerAssessment `json:"winner_assessment"`

	// Status of analytics engine
	Status []string `json:"status"`

	// Human-readable explanation of the status
	StatusInterpretations map[string]string `json:"status_interpretations"`

	// Last recorded state from analytics service
	LastState interface{} `json:"laste_state"`
}

// VersionAssessment contains assessment details for a version
type VersionAssessment struct {
	WinProbability       float32               `json:"win_probability"`
	RequestCount         int32                 `json:"request_count"`
	CriterionAssessments []CriterionAssessment `json:"criterion_assessments"`
}

// CriterionAssessment contains assessment for a version
type CriterionAssessment struct {
	// Id of version
	ID string `json:"id"`

	// ID of metric
	MetricID string `json:"metric_id"`

	//Statistics for this metric
	Statistics `json:"statistics"`

	// Assessment of how well this metric is doing with respect to threshold.
	// Defined only for metrics with a threshold
	ThresholdAssessment `json:"threshold_assessment,omitempty"`
}

// Statistics for a metric
type Statistics struct {
	Value           float32 `json:"value"`
	RatioStatistics `json:"ratio_statitics"`
}

// RatioStatistics is statistics for a ratio metric
type RatioStatistics struct {
	ImprovementOverBaseline       Interval `json:"improvement_over_baseline"`
	ProbabilityOfBeatingBaseline  float32  `json:"probability_of_beating_baseline"`
	ProbabilityOfBeingBestVersion float32  `json:"probability_of_being_best_version"`
	CredibleInterval              Interval `json:"credible_interval"`
}

// Interval for probability
type Interval struct {
	Lower float32 `json:"lower"`
	Upper float32 `json:"upper"`
}

// ThresholdAssessment is the assessment for a threshold
type ThresholdAssessment struct {
	// A flag indicating whether threshold is breached
	ThresholdBreached bool `json:"threshold_breached"`
	// Probability of satisfying the threshold.
	// Defined only for ratio metrics. This is currently computed based on Bayesian estimation
	ProbabilityOfSatisfyingTHreshold float32 `json:"probability_of_satisfying_threshold"`
}

// CandidateAssessment contains assessment for a candidate
type CandidateAssessment struct {
	VersionAssessment `json:",inline"`
	// A flag indicates whether traffic to this candidate should be cutoff
	Rollback bool `json:"rollback"`
}

// WinnerAssessment contains assessment for a winner version
type WinnerAssessment struct {
	// Indicates whether or not a clear winner has emerged
	// This is currently computed based on Bayesian estimation and uses posterior_probability_for_winner from the iteration parameters
	WinnerFound bool `json:"winning_version_found"`

	// ID of the current winner with the maximum probability of winning.
	// This is currently computed based on Bayesian estimation
	Winner string `json:"current_winner,omitempty"`

	//Posterior probability of the version declared as the current winner.
	// This is None if winner is None. This is currently computed based on Bayesian estimation
	Probability float32 `json:"winning_probability,omitempty"`
}