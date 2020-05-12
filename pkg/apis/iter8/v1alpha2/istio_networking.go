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

// This file contains re-typed HTTPMatchRequest from networking.istio.io/v1alpha3.
// CRD generator from sigs.k8s.io/controller-tools fails to recognize orginal format.

type HTTPMatchRequest struct {
	// The name assigned to a match.
	Name string `json:"name,omitempty"`

	// URI to match
	URI *StringMatch `json:"uri,omitempty"`

	// URI Scheme
	Scheme *StringMatch `json:"scheme,omitempty"`

	// HTTP Method
	Method *StringMatch `json:"method,omitempty"`

	// HTTP Authority
	Authority *StringMatch `json:"authority,omitempty"`

	// Headers to match
	Headers map[string]StringMatch `json:"headers,omitempty"`

	// Specifies the ports on the host that is being addressed.
	Port uint32 `json:"port,omitempty"`

	// SourceLabels for matching
	SourceLabels map[string]string `json:"sourceLabels,omitempty"`

	// Gateways for matching
	Gateways []string `json:"gateways,omitempty"`

	// Query parameters for matching.
	QueryParams map[string]StringMatch `json:"query_params,omitempty"`

	// Flag to specify whether the URI matching should be case-insensitive.
	IgnoreURICase bool `json:"ignore_uri_case,omitempty"`
}

type StringMatch struct {
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}
