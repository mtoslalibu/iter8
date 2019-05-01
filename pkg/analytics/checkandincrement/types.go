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

package checkandincrement

type Request struct {
	// Specifies a time interval and key-value pairs for retrieving and processing data pertaining to the baseline version
	Baseline Window `json:"baseline"`

	// Specifies a time interval and key-value pairs for retrieving and processing data pertaining to the canary version
	Canary Window `json:"canary"`

	// Parameters controlling the behavior of the analytics
	TrafficControl TrafficControl `json:"traffic_control"`

	// State returned by the server on the previous call
	LastState interface{} `json:"_last_state"`
}

type Window struct {
	// ISO8601 timestamp for the beginning of the time range of interest
	StartTime string `json:"start_time"`

	// ISO8601 timestamp for the end of the time range of interest; if omitted, current time is assumed
	EndTime string `json:"end_time"`

	// Key-value pairs identifying the data pertaining to a version
	Tags map[string]string `json:"tags"`
}

type TrafficControl struct {
	// Parameters controlling the behavior of the analytics
	WarmupRequestCount int `json:"warmup_request_count"`

	// Maximum percentage of traffic that the canary version will receive during the experiment; defaults to 50%
	MaxTrafficPercent float64 `json:"max_traffic_percent"`

	// Increment (in percent points) to be applied to the traffic received by the canary version
	// each time it passes the success criteria; defaults to 1 percent point
	StepSize float64 `json:"step_size"`

	// List of criteria for assessing the canary version
	SuccessCriteria []SuccessCriterion `json:"success_criteria"`
}

type SuccessCriterionType string

const (
	SuccessCriterionDelta     SuccessCriterionType = "delta"
	SuccessCriterionThreshold SuccessCriterionType = "threshold"
)

type SuccessCriterion struct {
	// Name of the metric to which the criterion applies
	// example: iter8_latency
	MetricName string `json:"metric_name"`

	// 	Criterion type. Options:
	// "delta": compares the canary against the baseline version with respect to the metric;
	// "threshold": checks the canary with respect to the metric
	Type SuccessCriterionType `json:"type"`

	// Determines how the traffic must be split at the end of the experiment; options:
	// "baseline": all traffic goes to the baseline version;
	// "canary": all traffic goes to the canary version;
	// "both": traffic is split across baseline and canary.
	// Defaults to “canary”
	OnSuccess string `json:"on_success"`

	// Value to check
	Value float64 `json:"value"`
}

type Response struct {
	// Measurements and traffic recommendation for the baseline version
	Baseline MetricsTraffic `json:"baseline"`

	// Measurements and traffic recommendation for the baseline version
	Canary MetricsTraffic `json:"canary"`
}

type MetricsTraffic struct {
	// Measurements and traffic recommendation for the baseline version
	TrafficPercentage float64 `json:traffic_percentage"`

	// Sate returned by the server, to be passed on the next call
	LastState interface{} `json:"_last_state"`
}
